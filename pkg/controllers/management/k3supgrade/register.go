package k3supgrade

import (
	"context"

	"github.com/rancher/rancher/pkg/clustermanager"
	"github.com/rancher/rancher/pkg/systemaccount"
	"github.com/rancher/rancher/pkg/wrangler"
	wranglerv3 "github.com/rancher/rancher/pkg/wrangler/generated/controllers/management.cattle.io/v3"
	v3 "github.com/rancher/types/apis/management.cattle.io/v3"
	v32 "github.com/rancher/types/apis/project.cattle.io/v3"
	"github.com/rancher/types/config"
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
