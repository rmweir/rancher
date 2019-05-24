package featureflags

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/rancher/norman/store/crd"
	"github.com/rancher/norman/types"
	managementschema "github.com/rancher/types/apis/management.cattle.io/v3/schema"
	"github.com/rancher/types/client/management/v3"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/util/feature"
)

const (
	Alpha string = "alpha"
	Beta  string = "beta"
	GA    string = "ga"
)

var (
	GlobalFeatures = newFeatureGate()
	FeaturePacks   = map[string]*FeaturePack{}

	KontainerDrivers = NewFeature(strings.ToLower(client.KontainerDriverType), "beta", true)
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
// TODO: setup enable to run staartup functions if it has not been started yet
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
			def,
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

type FeaturePack struct {
	Name       string
	Def        bool
	IsStarted  bool
	Crds       []string
	StartFuncs []interface{}
	StartArgs  [][]interface{}
	Collection Collection
	Schemas    *types.Schemas
}

type featStore struct {
	types.Store

	name string
}

type Collection interface {
	DeleteCollection(deleteOpts *v1.DeleteOptions, listOpts v1.ListOptions) error
}

func RunFeatureCRDS(factory *crd.Factory, ctx context.Context, storageContext types.StorageContext, schemas *types.Schemas, version *types.APIVersion) {
	enabledFeatureCRDS := []string{}
	g := GlobalFeatures.KnownFeatures()
	for _, name := range g {
		n := strings.Split(name, "=")[0]
		feat := feature.Feature(n)
		if GlobalFeatures.Enabled(feat) {
			f := FeaturePacks[n]
			enabledFeatureCRDS = append(enabledFeatureCRDS, f.Crds...)
		}
	}
	factory.BatchCreateCRDs(ctx, storageContext, schemas, version, enabledFeatureCRDS...)
}

func RunFeatureFns() {
	for _, name := range GlobalFeatures.KnownFeatures() {
		n := strings.Split(name, "=")[0]
		feat := feature.Feature(n)
		if GlobalFeatures.Enabled(feat) {
			fu := FeaturePacks
			for index, f := range fu[n].StartFuncs {
				args := FeaturePacks[n].StartArgs[index]
				runFunction(f, args)
			}
			FeaturePacks[n].start()
		}
	}
}

func (f *FeaturePack) start() {
	s := f.Schemas.Schema(&managementschema.Version, f.Name)
	s.Store = &featStore{
		s.Store,
		f.Name,
	}
	f.IsStarted = true
}

func runFunction(fn interface{}, args []interface{}) {
	val := reflect.ValueOf(fn)
	callArgs := make([]reflect.Value, len(args))
	for i, a := range args {
		callArgs[i] = reflect.ValueOf(a)
	}
	val.Call(callArgs)
}

func (f *FeaturePack) addStartFunc(fn interface{}) error {
	if reflect.TypeOf(fn).Kind() == reflect.Func {
		f.StartFuncs = append(f.StartFuncs, fn)
		return nil
	} else {
		return fmt.Errorf("Must add a function")
	}
}

func (f *FeaturePack) addCrds(crd string) {
	f.Crds = append(f.Crds, crd)
}

func (f *FeaturePack) Load() {
	FeaturePacks[f.Name] = f

}

// TODO probably delete
func (f *FeaturePack) Disable() {
	schema := f.Schemas.Schema(&managementschema.Version, f.Name)
	schema.Validator = nil
	schema.ActionHandler = nil
	schema.Formatter = nil
}

func (f *FeaturePack) Set(b string) error {
	return GlobalFeatures.Set(f.Name + "=" + b)
}

func Set(name string) error {
	if split := strings.Split(name, "="); len(split) > 1 {
		// FeaturePacks[split[0]].Disable()
		FeaturePacks[split[0]].Set(split[1])
	}
	return nil
}

// TODO probably delete
func (f *FeaturePack) Enable(name string) {
	GlobalFeatures.Set(name + "=true")
}

func (f *featStore) Create(apiContext *types.APIContext, schema *types.Schema, data map[string]interface{}) (map[string]interface{}, error) {
	feat := feature.Feature(f.name)
	if GlobalFeatures.Enabled(feat) {
		return f.Store.Create(apiContext, schema, data)
	}
	return nil, fmt.Errorf("TEST FEATURE disabled")
}
