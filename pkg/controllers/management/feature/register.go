package feature

import (
	"context"
	"fmt"
	"github.com/rancher/rancher/pkg/api/server/managementstored"
	"github.com/rancher/types/apis/management.cattle.io/v3"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime"
	"strings"

	"github.com/rancher/rancher/pkg/clustermanager"
	"github.com/rancher/types/config"
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
	if obj == nil || obj.DeletionTimestamp != nil {
		return nil, nil
	}

	logrus.Infof("TEST feat setting sync: %s", obj.Name)
	if split := strings.Split(obj.Name, "feat-"); len(split) > 1 {
		featureSet := obj.Value
		// TODO use feature packs sets
		if setting := strings.Split(obj.Value, "="); len(setting) > 1 {
			if m := managementstored.FeaturePacks[setting[0]]; m ==nil {
				return nil, fmt.Errorf("TEST FEATURE NIL %v", setting[1])
			} else {
				managementstored.Set(featureSet)
			}
			logrus.Info("TEST SUCCESS")
		}
	}

	return nil, nil
}
