package k3sUpgrade

import (
	"context"
	"fmt"

	"github.com/rancher/rancher/pkg/clustermanager"
	"github.com/rancher/rancher/pkg/wrangler"
	wranglerv3 "github.com/rancher/rancher/pkg/wrangler/generated/controllers/management.cattle.io/v3"
	v3 "github.com/rancher/types/apis/management.cattle.io/v3"
	"github.com/rancher/types/config"
	//"k8s.io/apimachinery/pkg/version"
)

type handler struct {
	systemUpgradeNamespace string
	newVersion             string //TODO make this info.Version
	clusterCache           wranglerv3.ClusterCache
}

func Register(ctx context.Context, wContext *wrangler.Context, mgmtCtx *config.ManagementContext, manager *clustermanager.Manager) error {

	h := &handler{
		systemUpgradeNamespace: "system-upgrade",
		newVersion:             "1.17.2+k3s",
		clusterCache:           wContext.Mgmt.Cluster().Cache(),
	}

	wContext.Mgmt.Cluster().OnChange(ctx, "k3s-upgrade-controller", h.onClusterChange)
	return nil
}

func (h *handler) onClusterChange(key string, cluster *v3.Cluster) (*v3.Cluster, error) {

	// only applies to k3s clusters
	if cluster.Status.Driver != v3.ClusterDriverK3s {
		return cluster, nil
	}

	fmt.Println("Cluster has changed OwO")
	return cluster, nil
}
