package k3supgrade

import (
	"context"

	"github.com/rancher/rancher/pkg/clustermanager"
	"github.com/rancher/rancher/pkg/systemaccount"
	"github.com/rancher/rancher/pkg/wrangler"
	wranglerv3 "github.com/rancher/rancher/pkg/wrangler/generated/controllers/management.cattle.io/v3"
	"github.com/rancher/system-upgrade-controller/pkg/generated/clientset/versioned/typed/upgrade.cattle.io/v1"
	v3 "github.com/rancher/types/apis/management.cattle.io/v3"
	v32 "github.com/rancher/types/apis/project.cattle.io/v3"
	"github.com/rancher/types/config"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type handler struct {
	systemUpgradeNamespace string
	newVersion             string //TODO make this info.Version
	clusterCache           wranglerv3.ClusterCache
	apps                   v32.AppInterface
	appLister              v32.AppLister
	templateLister         v3.CatalogTemplateLister
	systemAccountManager   *systemaccount.Manager
	manager                *clustermanager.Manager
}

const systemUpgradeNS = "system-upgrade"

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

func (h *handler) onClusterChange(key string, cluster *v3.Cluster) (*v3.Cluster, error) {

	if cluster == nil { //can this occur?
		return nil, nil
	}

	// only applies to k3s clusters
	if cluster.Status.Driver != v3.ClusterDriverK3s {
		return cluster, nil
	}

	// access downstream cluster
	clusterCtx, err := h.manager.UserContext(cluster.Name)
	if err != nil {
		return cluster, err
	}

	// create a client for GETing Plans in the downstream cluster
	// TODO: We shouldn't create one every time

	planClient, err := v1.NewForConfig(&clusterCtx.RESTConfig)
	if err != nil {
		return cluster, err
	}

	planList, err := planClient.Plans(h.systemUpgradeNamespace).List(metav1.ListOptions{})
	if err != nil {
		// may need to handle this error, if there is no Plan CRD what should we do?

		if errors.IsNotFound(err) {
			// no plan CRD exists
		}
		logrus.Warnf("err getting plans %s", err)
		return cluster, err
	}

	for plan := range planList.Items {
		fmt.Println("Found a plan")
		fmt.Printf("%+v\n", plan)
	}

	fmt.Println("Cluster has changed OwO")

	return cluster, nil
}

