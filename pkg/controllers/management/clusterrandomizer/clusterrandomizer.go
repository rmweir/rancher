package clusterrandomizer

import (
	"context"
	"math/rand"
	"strconv"
	"time"

	"github.com/rancher/types/apis/management.cattle.io/v3"
	"github.com/rancher/types/config"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
)

type Controller struct {
	cLister  v3.ClusterLister
	clusters v3.ClusterInterface
	rLister  v3.ClusterRandomizerLister
}

func Register(apiContext context.Context, management *config.ManagementContext) {
	c := &Controller{
		management.Management.Clusters("").Controller().Lister(),
		management.Management.Clusters(""),
		management.Management.ClusterRandomizers("").Controller().Lister(),
	}
}

func (c *Controller) Sync(key string, obj *v3.ClusterRandomizer) (runtime.Object, error) {
	clusters, err := c.cLister.List("", labels.Everything())
	if err != nil {
		return nil, err
	}

	for _, cluster := range clusters {
		clusterState := cluster.DeepCopy()
		rand.Seed(time.Now().Unix())
		clusterState.Spec.DisplayName = "cluster" + strconv.Itoa(rand.Intn(1000))
		if _, err := c.clusters.Update(clusterState); err != nil {
			return nil, err
		}
	}

	return nil, nil
}
