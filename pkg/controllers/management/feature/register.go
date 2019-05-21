package feature

import (
	"context"
	"encoding/json"
	"github.com/rancher/rancher/pkg/clustermanager"
	"github.com/rancher/rancher/pkg/features"
	"github.com/rancher/types/apis/management.cattle.io/v3"
	"github.com/rancher/types/config"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
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
	feature := featureflags.FeaturePacks[key]
	featureSet := key + "="
	featureMap := make(map[string]string)

	if feature == nil {
		return nil, nil
	}

	features, err := n.settings.Get("features", v1.GetOptions{})
	if err != nil {
		return nil, err
	}
	if features.Value != "" {
		err = json.Unmarshal([]byte(features.Value), featureMap)
		if err != nil {
			return nil, err
		}
	} else {
		err = json.Unmarshal([]byte(features.Default), &featureMap)
		if err != nil {
			return nil, err
		}
	}

	// If setting for feature is deleted, set feature to its default
	if obj == nil || obj.DeletionTimestamp != nil {
		if feature.Def {
			featureSet += "true"
			featureMap[key] = "true"
		} else {
			featureSet += "false"
			featureMap[key] = "false"
		}
	} else {
		featureSet += obj.Value
		featureMap[key] = obj.Value
	}

	b, err  := json.Marshal(featureMap)
	features.Value = string(b)
	n.settings.Update(features)
	featureflags.Set(featureSet)

	return nil, nil
}
