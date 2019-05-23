package example

import (
	"context"
	"fmt"
	"github.com/rancher/rancher/pkg/features"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"math/rand"
	"time"

	"github.com/rancher/types/apis/management.cattle.io/v3"
	"github.com/rancher/types/config"
)

type Controller struct {
	ctx                   context.Context
	clusterClient         v3.ClusterInterface
	clusterLister         v3.ClusterLister
}

func Register(ctx context.Context, management *config.ManagementContext) {
	c := &Controller{
		ctx:           ctx,
		clusterClient: management.Management.Clusters(""),
		clusterLister: management.Management.Clusters("").Controller().Lister(),
	}
	m := management.Management.ExampleConfigs("")
	s := &syn{
		c,
		"kontainerdrivers",
	}
	m.AddHandler(ctx, "example-controller", s.featureSync)
}

func (c *Controller) sync(key string, exampleConfig *v3.ExampleConfig) (runtime.Object, error) {
	fmt.Println("TEST IN")
	rand.Seed(time.Now().UTC().UnixNano())
	clusters, _ := c.clusterLister.List("", labels.Everything())
	for _, cluster := range clusters {
		cluster.Spec.DisplayName = fmt.Sprintf("cluster%v", rand.Intn(1000))
		_, err := c.clusterClient.Update(cluster)
		if err == nil {
			fmt.Println("TEST UPDATED SUCCESS")
		}
	}

	return nil, nil
}

func (s *syn) featureSync(key string, exampleConfig *v3.ExampleConfig) (runtime.Object, error) {
	if featureflags.GlobalFeatures.Enabled("kontainerdriver") {
		return s.sync(key, exampleConfig)
	}
	return nil, nil
}

type syncer interface {
	sync(string, *v3.ExampleConfig) (runtime.Object, error)
}

type syn struct {
	syncer
	feat string
}


