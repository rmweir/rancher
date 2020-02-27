package k3supgrade

import (
	"context"
	"fmt"

	"github.com/rancher/rancher/pkg/clustermanager"
	"github.com/rancher/rancher/pkg/systemaccount"
	"github.com/rancher/rancher/pkg/wrangler"
	wranglerv3 "github.com/rancher/rancher/pkg/wrangler/generated/controllers/management.cattle.io/v3"
	planClientset "github.com/rancher/system-upgrade-controller/pkg/generated/clientset/versioned/typed/upgrade.cattle.io/v1"
	v3 "github.com/rancher/types/apis/management.cattle.io/v3"
	projectv3 "github.com/rancher/types/apis/project.cattle.io/v3"
	"github.com/rancher/types/config"
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

const systemUpgradeNS = "system-upgrade"
const rancherManagedPlan = "rancher-managed"
const upgradeDisableLabelKey = "plan.upgrade.cattle.io/disable"

func Register(ctx context.Context, wContext *wrangler.Context, mgmtCtx *config.ManagementContext, manager *clustermanager.Manager) error {
	h := &handler{
		systemUpgradeNamespace: "system-upgrade",
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
func (h *handler) deployPlans(cluster v3.Cluster) error {

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
	planClient := planConfig.Plans(h.systemUpgradeNamespace)

	planList, err := planClient.List(metav1.ListOptions{})
	if err != nil {
		// may need to handle this error, if there is no Plan CRD what should we do?
		if errors.IsNotFound(err) {
			// no plan CRD exists
			logrus.Warnf("plan CRD does not exist: %s", err)
		}
		return err
	}

	// deactivate all existing plans that are not managed by Rancher
	for _, plan := range planList.Items {
		if _, ok := plan.Labels[rancherManagedPlan]; !ok {
			// inverse selection is used here, we select a non-existent label
			plan.Spec.NodeSelector.MatchExpressions = []metav1.LabelSelectorRequirement{{
				Key:      upgradeDisableLabelKey,
				Operator: metav1.LabelSelectorOpExists,
			}}

			_, err := planClient.Update(&plan)
			if err != nil {
				return err
			}
		}
	}

	// apply master and worker plans
	// TODO: what if they already exist?
	masterPlan, err := generateMasterPlan(cluster.Spec.K3sConfig.Version.String(), cluster.Spec.K3sConfig.ServerConcurrency)
	_, err = planClient.Create(&masterPlan)
	if err != nil {
		return err
	}
	workerPlan, err := generateWorkerPlan(cluster.Spec.K3sConfig.Version.String(),
		cluster.Spec.K3sConfig.WorkerConcurrency)
	_, err = planClient.Create(&workerPlan)
	if err != nil {
		return nil
	}
	fmt.Println("Deployed plans into cluster")

	return nil
}
