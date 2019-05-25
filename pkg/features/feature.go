package featureflags

import (
	"context"
	"fmt"
	"github.com/rancher/norman/httperror"

	//"github.com/rancher/norman/httperror"
	"reflect"
	"strings"

	"github.com/rancher/norman/store/crd"
	"github.com/rancher/norman/types"
	"github.com/rancher/rancher/pkg/clustermanager"
	managementschema "github.com/rancher/types/apis/management.cattle.io/v3/schema"
	"github.com/rancher/types/client/management/v3"
	"github.com/rancher/types/config"
	"github.com/sirupsen/logrus"
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

	KontainerDrivers = NewFeature(strings.ToLower(client.KontainerDriverType), "beta", false)
	ExampleConfig = NewFeature(strings.ToLower(client.ExampleConfigType), "beta", false)
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
		if FeaturePacks[n] != nil { // && GlobalFeatures.Enabled(feature.Feature(n)) {
			f := FeaturePacks[n]
			enabledFeatureCRDS = append(enabledFeatureCRDS, f.Crds...)
		}
	}

	if len(enabledFeatureCRDS) > 0 {
		factory.BatchCreateCRDs(ctx, storageContext, schemas, version, enabledFeatureCRDS...)
	}
}

func RunFeatureFns() {
	for _, name := range GlobalFeatures.KnownFeatures() {
		n := strings.Split(name, "=")[0]
		if FeaturePacks[n] != nil {
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
	origValidator := s.Validator
	s.Validator = func(types *types.APIContext, schema *types.Schema, m map[string]interface{}) error {
		feat := feature.Feature(f.Name)
		if !GlobalFeatures.Enabled(feat) {
			return httperror.NewAPIError(httperror.ActionNotAvailable, "TEST feature disabled")
		}
		return origValidator(types, schema, m)
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

func (f *FeaturePack) AddStartFunc(fn interface{}, args []interface{}) error {
	if reflect.TypeOf(fn).Kind() == reflect.Func {
		f.StartFuncs = append(f.StartFuncs, fn)
		f.StartArgs = append(f.StartArgs, args)
		return nil
	} else {
		return fmt.Errorf("Must add a function")
	}
}

func (f *FeaturePack) AddCrds(crd string) {
	f.Crds = append(f.Crds, crd)
}

func (f *FeaturePack) Load() {
	FeaturePacks[f.Name] = f

}

// TODO add disable functions to this
func (f *FeaturePack) Disable() {
	schema := f.Schemas.Schema(&managementschema.Version, f.Name)
	schema.Store = nil
	schema.ActionHandler = nil
}

func (f *FeaturePack) Set(b string) error {
	return GlobalFeatures.Set(f.Name + "=" + b)
}

func Set(name string) error {
	if split := strings.Split(name, "="); len(split) > 1 {
		FeaturePacks[split[0]].Set(split[1])
		if split[1] == "false" {
			// FeaturePacks[split[0]].Disable()
		} else {
			// FeaturePacks[split[0]].Enable()
		}
	}
	return nil
}

// TODO add start up functions to this
func (f *FeaturePack) Enable() {
	if f.IsStarted == false {
		f.start()
	}
	GlobalFeatures.Set(f.Name + "=true")
}

func (f *featStore) Create(apiContext *types.APIContext, schema *types.Schema, data map[string]interface{}) (map[string]interface{}, error) {
	feat := feature.Feature(f.name)
	if GlobalFeatures.Enabled(feat) {
		return f.Store.Create(apiContext, schema, data)
	}
	return nil, nil
}

func (f *featStore) Watch(apiContext *types.APIContext, schema *types.Schema, opts *types.QueryOptions) (chan map[string]interface{}, error) {
	feat := feature.Feature(f.name)
	if GlobalFeatures.Enabled(feat) {
		return f.Store.Watch(apiContext, schema, opts)
	}
	return nil, nil
}

func NewFeaturePack(name string, c Collection, ctx context.Context, apiContext *config.ScaledContext, clusterManager *clustermanager.Manager) *FeaturePack {
	f := feature.Feature(name)
	name = strings.ToLower(name)

	kd := &FeaturePack{
		name,
		GlobalFeatures.Enabled(f),
		false,
		[]string{},
		[]interface{}{},
		[][]interface{}{},
		apiContext.Schemas,
	}
	if kd.Def == false {
		logrus.Info("TEST DELETE COLLECTION")
	}
	kd.Load()

	return kd
}

func IsFeatEnabled(feat string) bool {
	a := GlobalFeatures.Enabled(feature.Feature(feat))
	return a
}