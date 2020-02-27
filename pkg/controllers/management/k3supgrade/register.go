package k3supgrade

import (
	"context"
	"fmt"
	"reflect"

	"github.com/rancher/rancher/pkg/clustermanager"
	"github.com/rancher/rancher/pkg/systemaccount"
	"github.com/rancher/rancher/pkg/wrangler"
	wranglerv3 "github.com/rancher/rancher/pkg/wrangler/generated/controllers/management.cattle.io/v3"
	planv1 "github.com/rancher/system-upgrade-controller/pkg/apis/upgrade.cattle.io/v1"
	planClientset "github.com/rancher/system-upgrade-controller/pkg/generated/clientset/versioned/typed/upgrade.cattle.io/v1"
	v3 "github.com/rancher/types/apis/management.cattle.io/v3"
	projectv3 "github.com/rancher/types/apis/project.cattle.io/v3"
	"github.com/rancher/types/config"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type handler struct {
	systemUpgradeNamespace string
	newVersion             string //TODO make this info.Version
	clusterCache           wranglerv3.ClusterCache
	apps                   projectv3.AppInterface
	appLister              projectv3.AppLister
	templateLister         v3.CatalogTemplateLister
	systemAccountManager   *systemaccount.Manager
	manager                *clustermanager.Manager
}

const systemUpgradeNS = "cattle-system"
const rancherManagedPlan = "rancher-managed"
const upgradeDisableLabelKey = "plan.upgrade.cattle.io/disable"

func Register(ctx context.Context, wContext *wrangler.Context, mgmtCtx *config.ManagementContext, manager *clustermanager.Manager) error {
	h := &handler{
		systemUpgradeNamespace: systemUpgradeNS,
		newVersion:             "1.17.2+k3s",
		clusterCache:           wContext.Mgmt.Cluster().Cache(),
		apps:                   mgmtCtx.Project.Apps(metav1.NamespaceAll),
		appLister:              mgmtCtx.Project.Apps("").Controller().Lister(),
		templateLister:         mgmtCtx.Management.CatalogTemplates("").Controller().Lister(),
		systemAccountManager:   systemaccount.NewManager(mgmtCtx),
		manager:                manager,
	}

	wContext.Mgmt.Cluster().OnChange(ctx, "k3s-upgrade-controller", h.onClusterChange)
	return nil
}

// deployPlans creates a master and worker plan in the downstream cluster to instrument
// the system-upgrade-controller in the downstream cluster
func (h *handler) deployPlans(cluster *v3.Cluster) error {

	// access downstream cluster
	clusterCtx, err := h.manager.UserContext(cluster.Name)
	if err != nil {
		return err
	}

	// create a client for GETing Plans in the downstream cluster
	// TODO: We shouldn't create one every time
	planConfig, err := planClientset.NewForConfig(&clusterCtx.RESTConfig)
	if err != nil {
		return err
	}
	planClient := planConfig.Plans(metav1.NamespaceAll)

	planList, err := planClient.List(metav1.ListOptions{})
	if err != nil {
		// may need to handle this error, if there is no Plan CRD what should we do?
		if errors.IsNotFound(err) {
			// no plan CRD exists
			logrus.Warnf("plan CRD does not exist: %s", err)
		}
		return err
	}
	masterPlan := planv1.Plan{}
	workerPlan := planv1.Plan{}
	// deactivate all existing plans that are not managed by Rancher
	for _, plan := range planList.Items {
		if _, ok := plan.Labels[rancherManagedPlan]; !ok {
			// inverse selection is used here, we select a non-existent label
			plan.Spec.NodeSelector = &metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{{
					Key:      upgradeDisableLabelKey,
					Operator: metav1.LabelSelectorOpExists,
				}}}

			_, err = planClient.Update(&plan)
			if err != nil {
				return err
			}
		} else {
			// if any of the rancher plans are currently applying, set updating status on cluster
			if len(plan.Status.Applying) > 0 {
				v3.ClusterConditionUpdated.True(cluster)
			}

			switch name := plan.Name; name {
			case k3sMasterPlanName:
				masterPlan = plan
			case k3sWorkerPlanName:
				workerPlan = plan
			}
		}
	}

	// if rancher plans exist, do we need to update?
	if masterPlan.Name != "" || workerPlan.Name != "" {
		if masterPlan.Name != "" {
			newMaster, err := configureMasterPlan(masterPlan, cluster.Spec.K3sConfig.Version.String(), cluster.Spec.K3sConfig.ServerConcurrency)
			if err != nil {
				return err
			}
			if !cmp(masterPlan, newMaster) {
				_, err = planClient.Update(&newMaster)
				if err != nil {
					return err
				}
			} else {
				fmt.Println("master plan is the same, not updating")
				// if we were in an updating state, flip back
				if v3.ClusterConditionUpdated.IsTrue(cluster) {
					v3.ClusterConditionUpdated.False(cluster)
				}
			}
		}

		if workerPlan.Name != "" {
			newWorker, err := configureWorkerPlan(workerPlan, cluster.Spec.K3sConfig.Version.String(), cluster.Spec.K3sConfig.WorkerConcurrency)
			if err != nil {
				return err
			}
			if !cmp(workerPlan, newWorker) {
				_, err = planClient.Update(&newWorker)
				if err != nil {
					return nil
				}
			} else {
				fmt.Println("worker plan is the same, not updating")
				// if we were in an updating state, flip back
				if v3.ClusterConditionUpdated.IsTrue(cluster) {
					v3.ClusterConditionUpdated.False(cluster)
				}
			}
		}

	} else { // create the plans
		masterPlan, err = generateMasterPlan(cluster.Spec.K3sConfig.Version.String(),
			cluster.Spec.K3sConfig.ServerConcurrency)
		_, err = planClient.Create(&masterPlan)
		if err != nil {
			return err
		}
		workerPlan, err = generateWorkerPlan(cluster.Spec.K3sConfig.Version.String(),
			cluster.Spec.K3sConfig.WorkerConcurrency)
		_, err = planClient.Create(&workerPlan)
		if err != nil {
			return nil
		}
		fmt.Println("Deployed plans into cluster")
	}

	return nil
}

//cmp compares two plans but does not compare their Status, returns true if they are the same
func cmp(a, b planv1.Plan) bool {
	if a.Name != b.Name {
		return false
	}
	if a.Namespace != b.Namespace {
		return false
	}

	if a.Spec.Version != b.Spec.Version {
		return false
	}

	if a.Spec.Concurrency != b.Spec.Concurrency {
		return false
	}

	//TODO Refactor to not use reflection
	if !reflect.DeepEqual(a.Spec, b.Spec) {
		return false
	}
	if !reflect.DeepEqual(a.ObjectMeta, b.ObjectMeta) {
		return false
	}
	if !reflect.DeepEqual(a.TypeMeta, b.TypeMeta) {
		return false
	}
	return true
}
