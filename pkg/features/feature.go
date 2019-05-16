package featureflags

import (
	"fmt"
	"k8s.io/apiserver/pkg/util/feature"
	"reflect"
)

const (
	Alpha string = "alpha"
	Beta string = "beta"
	GA string = "ga"
)

var (
	GlobalFeatures       	 = newFeatureGate()
	FeaturePackMap			 = map[string]featurePack{}

	KontainerDrivers = NewFeature("kontainerDrivers", "ga", true)
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
			def,
			feature.Alpha,
		}
	case Beta:
		f = &feature.FeatureSpec{
			true,
			feature.Beta,
		}
	case GA:
		f = &feature.FeatureSpec{
			def,
			feature.GA,
		}
	}
	featureName := feature.Feature(name)
	GlobalFeatures.Add(map[feature.Feature]feature.FeatureSpec{featureName: *f})

	g := GlobalFeatures
	if g == nil {

	}
	return nil
}


type featurePack struct {
	name string
	crds []string
	startFuncs []interface{}
	args [][]interface{}
}

func (f *featurePack) addStartFunc(fn interface{}) error {
	if reflect.TypeOf(fn).Kind() == reflect.Func {
		f.startFuncs = append(f.startFuncs, fn)
		return nil
	} else {
		return fmt.Errorf("Must add a function")
	}
}

func (f *featurePack) load() {
	FeaturePackMap[f.name] = *f
}

func (f *featurePack) addCrds(crd string) {
	f.crds = append(f.crds, crd)
}
