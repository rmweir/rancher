package nodetemplate

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"strings"

	"github.com/rancher/rancher/pkg/controllers/management/globalnamespacerbac"
	"github.com/rancher/rancher/pkg/namespace"
	"github.com/rancher/types/apis/management.cattle.io/v3"
	"github.com/rancher/types/config"
	"github.com/sirupsen/logrus"
)

const (
	NormanIDAnno = "cattle.io/creator"
)

type nodeTemplateController struct {
	roleLister v3.GlobalRoleLister
	roleClient v3.GlobalRoleInterface
	rbLister   v3.GlobalRoleBindingLister
	rbClient   v3.GlobalRoleBindingInterface
	ntClient   v3.NodeTemplateInterface
	ntLister   v3.NodeTemplateLister
	npLister   v3.NodePoolLister
	npClient   v3.NodePoolInterface
	mgmtCtx    *config.ManagementContext
}

func Register(ctx context.Context, mgmt *config.ManagementContext) {
	nt := nodeTemplateController{
		roleLister: mgmt.Management.GlobalRoles("").Controller().Lister(),
		roleClient: mgmt.Management.GlobalRoles(""),
		rbLister:   mgmt.Management.GlobalRoleBindings("").Controller().Lister(),
		rbClient:   mgmt.Management.GlobalRoleBindings(""),
		ntClient:   mgmt.Management.NodeTemplates(""),
		ntLister:   mgmt.Management.NodeTemplates("").Controller().Lister(),
		npLister:   mgmt.Management.NodePools("").Controller().Lister(),
		npClient:   mgmt.Management.NodePools(""),
		mgmtCtx:    mgmt,
	}

	mgmt.Management.NodeTemplates("").Controller().AddHandler(ctx, "nt-grb-handler", nt.sync)
}

func (nt *nodeTemplateController) sync(key string, nodeTemplate *v3.NodeTemplate) (runtime.Object, error) {
	if nodeTemplate == nil || nodeTemplate.DeletionTimestamp != nil {
		return nil, nil
	}

	// migration logic

	metaAccessor, err := meta.Accessor(nodeTemplate)
	if err != nil {
		return nodeTemplate, err
	}

	creatorID, ok := metaAccessor.GetAnnotations()[globalnamespacerbac.CreatorIDAnn]
	if !ok {
		return nodeTemplate, fmt.Errorf("clusterTemplate %v has no creatorId annotation", metaAccessor.GetName())
	}

	restConfig := nt.mgmtCtx.RESTConfig
	dynamicClient, err := dynamic.NewForConfig(&restConfig)
	if err != nil {
		return nil, err
	}

	s := schema.GroupVersionResource{
		Group:    "management.cattle.io",
		Version:  "v3",
		Resource: "nodetemplates",
	}

	ntDynamicClient := dynamicClient.Resource(s)
	migratedNTName := fmt.Sprintf("nt-%s-%s", nodeTemplate.Namespace, nodeTemplate.Name)
	if nodeTemplate.Namespace == creatorID && nodeTemplate.Labels[NormanIDAnno] == "norman" {
		if nodeTemplate.Annotations["migrated"] != "true" {
			// node template has not been fully migrated - duplicate user namespace node template in cattle-global-data namespace
			logrus.Infof("migrating node template [%s]", nodeTemplate.Spec.DisplayName)

			fullLegacyNTName := fmt.Sprintf("%s:%s", nodeTemplate.Namespace, nodeTemplate.Name)
			fullGlobalNTName := fmt.Sprintf("cattle-global-data:%s", migratedNTName)

			dynamicNodeTemplate, err := ntDynamicClient.Namespace(nodeTemplate.Namespace).Get(nodeTemplate.Name, metav1.GetOptions{})
			if err != nil {
				return nil, err
			}

			if err := nt.createGlobalNodeTemplateClone(nodeTemplate.Name, migratedNTName, dynamicNodeTemplate, ntDynamicClient); err != nil {
				return nil, err
			}

			if err := nt.reviseNodePoolNodeTemplate(fullGlobalNTName, fullLegacyNTName); err != nil {
				return nil, err
			}

			if err := nt.reviseNodes(fullGlobalNTName, fullLegacyNTName); err != nil {
				return nil, err
			}

			legacyAnnotations, err := getDynamicAnnotations(dynamicNodeTemplate, fullLegacyNTName)
			if err != nil {
				return nil, err
			}

			legacyAnnotations["migrated"] = "true"
			dynamicNodeTemplate.Object["annotations"] = legacyAnnotations

			_, err = dynamicClient.Resource(s).Namespace(nodeTemplate.Namespace).Update(dynamicNodeTemplate, metav1.UpdateOptions{})
			if err != nil {
				return nil, err
			}

			// the annotation has been updated via the dynamic client, so the update node template should be fetch and returned
			nodeTemplate, err = nt.ntClient.Controller().Lister().Get(nodeTemplate.Namespace, nodeTemplate.Name)
			if err != nil {
				return nil, err
			}

			logrus.Infof("successfully migrated node template [%s]", nodeTemplate.Spec.DisplayName)
		}
	} else {
		dynamicNodeTemplate, err := ntDynamicClient.Namespace(nodeTemplate.Namespace).Get(nodeTemplate.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}

		annotations, err := getDynamicAnnotations(dynamicNodeTemplate, migratedNTName)
		if err != nil {
			return nil, err
		}

		if annotations["ownerBindingsCreated"] != "true" {
			// Create Role and RBs if they do not exist
			if err := globalnamespacerbac.CreateRoleAndRoleBinding(globalnamespacerbac.NodeTemplateResource, nodeTemplate.Name,
				globalnamespacerbac.RancherManagementAPIVersion, creatorID, []string{globalnamespacerbac.RancherManagementAPIVersion},
				nodeTemplate.UID,
				[]v3.Member{}, nt.mgmtCtx); err != nil {
				return nil, err
			}

			dynamicNodeTemplate, err := writeDynamimcAnnotations(dynamicNodeTemplate, migratedNTName, "ownerBindingsCreated", "true")
			if err != nil {
				return nil, err
			}

			if _, err := ntDynamicClient.Namespace(nodeTemplate.Namespace).Create(dynamicNodeTemplate, metav1.CreateOptions{}); err != nil {
				return nil, err
			}

			nodeTemplate, err = nt.ntClient.Controller().Lister().Get(nodeTemplate.Namespace, nodeTemplate.Name)
			if err != nil {
				return nil, err
			}
		}
	}

	// intentionally returning nil, as node template should be retrieved via dynamic client
	return nodeTemplate, nil
}

func (nt *nodeTemplateController) reviseNodePoolNodeTemplate(fullGlobalNTName, fullLegacyNTName string) error {
	npList, err := nt.npLister.List("", labels.Everything())
	if err != nil {
		return err
	}
	for _, np := range npList {
		if np.Spec.NodeTemplateName == fullLegacyNTName {
			npCopy := np.DeepCopy()
			npCopy.Spec.NodeTemplateName = fullGlobalNTName

			_, err := nt.npClient.Update(npCopy)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (nt *nodeTemplateController) reviseNodes(fullGlobalNTName, fullLegacyNTName string) error {
	nodeList, err := nt.mgmtCtx.Management.Nodes("").Controller().Lister().List("", labels.Everything())
	if err != nil {
		return err
	}
	for _, node := range nodeList {
		if node.Spec.NodeTemplateName == fullLegacyNTName {
			nodeCopy := node.DeepCopy()
			nodeCopy.Spec.NodeTemplateName = fullGlobalNTName

			_, err := nt.mgmtCtx.Management.Nodes("").Update(nodeCopy)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func getDynamicAnnotations(dynamicNodeTemplate *unstructured.Unstructured, nodeTemplateName string) (map[string]interface{}, error) {
	metadata, ok := dynamicNodeTemplate.Object["metadata"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("error fetching node template [%s] metadata", nodeTemplateName)
	}

	annotations, ok := metadata["annotations"].(map[string]interface{})
	if !ok {
		annotations = make(map[string]interface{})
	}

	return annotations, nil
}

func writeDynamimcAnnotations(dynamicNodeTemplate *unstructured.Unstructured, nodeTemplateName string, key string, value string) (*unstructured.Unstructured, error) {
	metadata, ok := dynamicNodeTemplate.Object["metadata"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("error fetching node template [%s] metadata", nodeTemplateName)
	}

	annotations, ok := metadata["annotations"].(map[string]interface{})
	if !ok {
		annotations = make(map[string]interface{})
	}

	annotations[key] = value
	metadata["annotations"] = annotations

	dynamicNodeTemplate.Object["metadata"] = metadata

	return dynamicNodeTemplate, nil
}

// createGlobalNodeTemplateClone returns the global clone of the given legacy node templates. If one does not exist
// it will be created
func (nt *nodeTemplateController) createGlobalNodeTemplateClone(legacyName, cloneName string, dynamicNodeTemplate *unstructured.Unstructured, client dynamic.NamespaceableResourceInterface) (error) {
	_, err := nt.ntLister.Get("cattle-global-data", cloneName)
	if err != nil {
		if !strings.Contains(err.Error(), "not found") {
			return err
		}

		globalNodeTemplate := dynamicNodeTemplate.DeepCopy()

		annotations, err := getDynamicAnnotations(dynamicNodeTemplate, legacyName)
		if err != nil {
			return err
		}

		globalNodeTemplate.Object["metadata"] = map[string]interface{}{
			"name":        cloneName,
			"namespace":   namespace.GlobalNamespace,
			"annotations": annotations,
		}

		globalNodeTemplate, err = client.Namespace("cattle-global-data").Create(globalNodeTemplate, metav1.CreateOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

