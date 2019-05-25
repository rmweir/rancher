package app

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/rancher/norman/httperror"
	featureflags "github.com/rancher/rancher/pkg/features"
	"github.com/rancher/rancher/pkg/settings"
)

func addFeature(cfg Config) error {
	featureMap := make(map[string]string)

	features := settings.Features.Get()

	if features != "" {
		err := json.Unmarshal([]byte(features), &featureMap)
		if err != nil {
			return fmt.Errorf("unable to read features setting in add feature data")
		}
	}

	argFeatures := strings.Split(cfg.Features, ",")
	for _, f := range argFeatures {
		parts := strings.Split(f, "=")
		if len(parts) != 2 {
			return httperror.NewAPIError(httperror.InvalidBodyContent, "features value must be in \"featureName=boolValue\" format")
		}
		name := parts[0]
		value := parts[1]
		if featPack, ok := featureflags.FeaturePacks[name]; ok {
			featPack.Set(value)
			featureMap[name] = value
		}
	}

	b, err := json.Marshal(featureMap)
	if err != nil {
		return fmt.Errorf("unable to convert feature map to btes in add feature data")
	}
	settings.Features.Set(string(b))

	return nil
}
