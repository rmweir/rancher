package projectsetter

import (
	"github.com/rancher/norman/store/transform"
	"github.com/rancher/norman/types"
	"github.com/rancher/norman/types/convert"
	"github.com/rancher/rancher/pkg/clustermanager"
	"github.com/rancher/types/apis/core/v1"
	"github.com/rancher/types/client/cluster/v3"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/labels"
	"strings"
)

func New(store types.Store, manager *clustermanager.Manager) types.Store {
	t := &transformer{
		ClusterManager: manager,
	}
	return &transform.Store{
		Store:             store,
		Transformer:       t.object,
		ListTransformer:   t.list,
		StreamTransformer: t.stream,
		OptionsTransformer: t.options,
	}
}

type transformer struct {
	ClusterManager *clustermanager.Manager
}

func (t *transformer) object(apiContext *types.APIContext, schema *types.Schema, data map[string]interface{}, opt *types.QueryOptions) (map[string]interface{}, error) {
	t.lookupAndSetProjectID(apiContext, schema, data)
	return data, nil
}

func (t *transformer) list(apiContext *types.APIContext, schema *types.Schema, data []map[string]interface{}, opt *types.QueryOptions) ([]map[string]interface{}, error) {
	if apiContext.Type == "project" {
		logrus.Info("TEST project opts")
	}
	t.options(apiContext, schema, opt)
	namespaceLister := t.lister(apiContext, schema)
	if namespaceLister == nil {
		return data, nil
	}

	for _, item := range data {
		setProjectID(namespaceLister, item)
	}

	// t.setProjectNamespaceOpts(apiContext, schema, data, opt)

	return data, nil
}

func (t *transformer) options(apiContext *types.APIContext, schema *types.Schema, opt *types.QueryOptions) error {
	if apiContext.Type == "project" {
		logrus.Info("TEST project opts")
	}
	if apiContext.SubContext == nil || apiContext.SubContext["/v3/schemas/project"] == "" {
		return nil
	}
	projectID := strings.Split(apiContext.SubContext["/v3/schemas/project"], ":")[1]

	clusterName := t.ClusterManager.ClusterName(apiContext)
	if clusterName == "" {
		return nil
	}

	clusterContext, err := t.ClusterManager.UserContext(clusterName)
	if err != nil {
		return err
	}

	namespaces, err := clusterContext.Core.Namespaces("").Controller().Lister().List("", labels.NewSelector())
	if err != nil {
		return err
	}

	var nsKeys []string
	for _, ns := range namespaces {
		if strings.Split(ns.Annotations["field.cattle.io/projectId"], ":")[1] == projectID {
			nsKeys = append(nsKeys, ns.Name)

		}

	}

	opt.Conditions = append(opt.Conditions, types.NewConditionFromString("namespaceId", "eq", nsKeys...))
	return nil
}

func (t *transformer) stream(apiContext *types.APIContext, schema *types.Schema, data chan map[string]interface{}, opt *types.QueryOptions) (chan map[string]interface{}, error) {
	namespaceLister := t.lister(apiContext, schema)
	if namespaceLister == nil {
		return data, nil
	}

	return convert.Chan(data, func(data map[string]interface{}) map[string]interface{} {
		setProjectID(namespaceLister, data)
		return data
	}), nil
}

func (t *transformer) lister(apiContext *types.APIContext, schema *types.Schema) v1.NamespaceLister {
	if apiContext.Type == "projects" || apiContext.Type == "project" {
		logrus.Info("TEST PODS LISTER TRANSFORM")
	}
	if _, ok := schema.ResourceFields[client.NamespaceFieldProjectID]; !ok || schema.ID == client.NamespaceType {
		return nil
	}

	clusterName := t.ClusterManager.ClusterName(apiContext)
	if clusterName == "" {
		return nil
	}

	clusterContext, err := t.ClusterManager.UserContext(clusterName)
	if err != nil {
		return nil
	}

	return clusterContext.Core.Namespaces("").Controller().Lister()
}

func (t *transformer) lookupAndSetProjectID(apiContext *types.APIContext, schema *types.Schema, data map[string]interface{}) {
	namespaceLister := t.lister(apiContext, schema)
	if namespaceLister == nil {
		return
	}

	setProjectID(namespaceLister, data)
}

func setProjectID(namespaceLister v1.NamespaceLister, data map[string]interface{}) {
	if data == nil {
		return
	}

	ns, _ := data["namespaceId"].(string)
	projectID, _ := data[client.NamespaceFieldProjectID].(string)
	if projectID != "" {
		return
	}

	nsObj, err := namespaceLister.Get("", ns)
	if err != nil {
		return
	}

	data[client.NamespaceFieldProjectID] = nsObj.Annotations["field.cattle.io/projectId"]
}
