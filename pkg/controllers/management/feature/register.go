package feature

import (
	"context"
	"fmt"
	"github.com/rancher/rancher/pkg/api/server/managementstored"
	"github.com/rancher/rancher/pkg/clustermanager"
	"github.com/rancher/types/apis/management.cattle.io/v3"
	"github.com/rancher/types/config"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	featSettingController = "feat-kontainer-driver"
)
func RegisterEarly(ctx context.Context, management *config.ManagementContext, clusterManager *clustermanager.Manager) {
	s := newFeatSettingController(management)

	management.Management.Settings("").AddHandler(ctx, featSettingController, s.sync)
}


type SettingController struct {
	settings v3.SettingInterface
}

func newFeatSettingController(mgmt *config.ManagementContext) *SettingController {
	n := &SettingController{
		settings: mgmt.Management.Settings(""),
	}
	return n
}

//sync is called periodically and on real updates
func (n *SettingController) sync(key string, obj *v3.Setting) (runtime.Object, error) {
	feature := managementstored.FeaturePacks[key]
	featureSet := key + "="

	if feature == nil {
		return nil, fmt.Errorf("TEST FEATURE NIL %v", key)
	}

	// If setting for feature is deleted, set feature to its default
	if obj == nil || obj.DeletionTimestamp != nil {
		if feature.Def {
			featureSet += "true"
		} else {
			featureSet += "false"
		}
	} else {
		featureSet += obj.Value
	}

	managementstored.Set(featureSet)

	return nil, nil
}
