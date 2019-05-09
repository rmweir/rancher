package featureflags

import (
	"k8s.io/apiserver/pkg/util/feature"
)

const (
	Alpha string = "alpha"
	Beta string = "beta"
	GA string = "ga"
)

var (
	GlobalFeatures       	 = newFeatureGate()

	KontainerDrivers = NewFeature("kontainerDriver", "alpha", false)
)

type FeatureGate interface {
	// Set parses and stores flag gates for known features
	// from a string like feature1=true,feature2=false,...
	Set(value string) error
	// SetFromMap stores flag gates for known features from a map[string]bool or returns an error
	SetFromMap(m map[string]bool) error
	// Enabled returns true if the key is enabled.
	Enabled(key feature.Feature) bool
	// Add adds features to the featureGate.
	Add(features map[feature.Feature]feature.FeatureSpec) error
	// KnownFeatures returns a slice of strings describing the FeatureGate's known features.
	KnownFeatures() []string
	// DeepCopy returns a deep copy of the FeatureGate object, such that gates can be
	// set on the copy without mutating the original. This is useful for validating
	// config against potential feature gate changes before committing those changes.
}

func newFeatureGate() FeatureGate {
	return feature.NewFeatureGate()
}

func init() {

}

func NewFeature(name string, release string, def bool) *feature.FeatureSpec {
	var f *feature.FeatureSpec

	switch release {
	case Alpha:
		f = &feature.FeatureSpec{
			false,
			feature.Alpha,
		}
	case Beta:
		f = &feature.FeatureSpec{
			true,
			feature.Beta,
		}
	case GA:
		f = &feature.FeatureSpec{
			true,
			feature.GA,
		}
	}
	featureName := feature.Feature(name)
	features.Add(map[feature.Feature]feature.FeatureSpec{featureName: *f})

	return nil
}
