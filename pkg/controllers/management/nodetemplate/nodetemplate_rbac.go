package nodetemplate

import (
	"context"
	"fmt"
	"strings"

	k8srbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	"github.com/rancher/rancher/pkg/controllers/management/globalnamespacerbac"
	"github.com/rancher/rancher/pkg/namespace"
	v3 "github.com/rancher/types/apis/management.cattle.io/v3"
	"github.com/rancher/types/config"
	"github.com/sirupsen/logrus"
)

const (
	NormanIDAnno = "cattle.io/creator"
	ctLabel      = "io.cattle.field/clusterTemplateId"
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

	// Duplicate user namespace node template
	if nodeTemplate.Namespace == creatorID && nodeTemplate.Labels[NormanIDAnno] == "norman" {
		if nodeTemplate.Annotations["migrated"] != "true" {
			logrus.Infof("migrating node template [%s]", nodeTemplate.Spec.DisplayName)
			migratedNTName := fmt.Sprintf("nt-%s-%s", nodeTemplate.Namespace, nodeTemplate.Name)

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

			dynamicNodeTemplate, err := dynamicClient.Resource(s).Namespace(nodeTemplate.Namespace).Get(nodeTemplate.Name, metav1.GetOptions{})
			if err != nil {
				return nil, err
			}
			globalNodeTemplate, err := dynamicClient.Resource(s).Namespace("cattle-global-data").Get(migratedNTName, metav1.GetOptions{})
			if err != nil {

				// legacy template has not been created yet, create it
				if !strings.Contains(err.Error(), "not found") {
					return nil, err
				}

				globalNodeTemplate = dynamicNodeTemplate.DeepCopy()
				globalNodeTemplate.Object["metadata"] = map[string]interface{}{
					"name":        migratedNTName,
					"namespace":   namespace.GlobalNamespace,
					"annotations": nodeTemplate.Annotations,
				}

				globalNodeTemplate, err = dynamicClient.Resource(s).Namespace("cattle-global-data").Create(globalNodeTemplate, metav1.CreateOptions{})
				if err != nil {
					return nil, err
				}
			}

			fullGlobalNTName := fmt.Sprintf("cattle-global-data:%s", globalNodeTemplate.Object["name"])
			npList, err := nt.npLister.List("", labels.Everything())
			if err != nil {
				return nil, err
			}
			for _, np := range npList {
				if np.Spec.NodeTemplateName == fmt.Sprintf("%s:%s", nodeTemplate.Namespace, nodeTemplate.Name) {
					npCopy := np.DeepCopy()
					npCopy.Spec.NodeTemplateName = fullGlobalNTName

					_, err := nt.npClient.Update(npCopy)
					if err != nil {
						return nil, err
					}
				}
			}

			nodeList, err := nt.mgmtCtx.Management.Nodes("").Controller().Lister().List("", labels.Everything())
			if err != nil {
				return nil, err
			}
			for _, node := range nodeList {
				if node.Spec.NodeTemplateName == fmt.Sprintf("%s:%s", nodeTemplate.Namespace, nodeTemplate.Name) {
					nodeCopy := node.DeepCopy()
					nodeCopy.Spec.NodeTemplateName = fullGlobalNTName

					_, err := nt.mgmtCtx.Management.Nodes("").Update(nodeCopy)
					if err != nil {
						return nil, err
					}
				}
			}

			annotations, _ := dynamicNodeTemplate.Object["annotations"].(map[string]interface{})
			annotations["migrated"] = "true"
			dynamicNodeTemplate.Object["annotations"] = annotations
			globalNodeTemplate, err = dynamicClient.Resource(s).Namespace(nodeTemplate.Namespace).Create(dynamicNodeTemplate, metav1.CreateOptions{})
			if err != nil {
				return nil, err
			}

			_, err = nt.ntClient.Update(nodeTemplate)
			if err != nil {
				return nil, err
			}

			// the annotation has been updated via the dynamic client, so the update node template should be fetch and returned
			nodeTemplate, err := nt.ntClient.Controller().Lister().Get(nodeTemplate.Namespace, nodeTemplate.Name)
			if err != nil {
				return nil, err
			}
			logrus.Infof("successfully migrated node template [%s]", nodeTemplate.Spec.DisplayName)
		}
	}

	// Create Role and RBs
	if err := globalnamespacerbac.CreateRoleAndRoleBinding(globalnamespacerbac.NodeTemplateResource, nodeTemplate.Name,
		globalnamespacerbac.RancherManagementAPIVersion, creatorID, []string{globalnamespacerbac.RancherManagementAPIVersion},
		nodeTemplate.UID,
		[]v3.Member{}, nt.mgmtCtx); err != nil {
		return nil, err
	}

	// intentionally returning nil, as node template should be retrieved via dynamic client
	return nodeTemplate, nil
}

func (nt *nodeTemplateController) createRole(nodeTemplate *v3.NodeTemplate, ownerRef metav1.OwnerReference) (*v3.GlobalRole, error) {
	roleName := "grb-nt-" + nodeTemplate.Name + "-" + nodeTemplate.Annotations[globalnamespacerbac.CreatorIDAnn]
	ntRole, err := nt.roleLister.Get("", roleName)
	if err != nil {
		if !strings.Contains(err.Error(), "not found") {
			return nil, err
		}
		newRole := &v3.GlobalRole{
			ObjectMeta: metav1.ObjectMeta{
				Name:            roleName,
				OwnerReferences: []metav1.OwnerReference{ownerRef},
			},
			Rules: []k8srbacv1.PolicyRule{
				{
					APIGroups:     []string{globalnamespacerbac.RancherManagementAPIVersion},
					Resources:     []string{"nodetemplates"},
					ResourceNames: []string{nodeTemplate.Name},
					Verbs:         []string{"*"},
				},
			},
		}
		return nt.roleClient.Create(newRole)
	}
	return ntRole, nil
}

func (nt *nodeTemplateController) createGRB(user, roleName string) (*v3.GlobalRoleBinding, error) {
	name := "grb-nt-" + roleName + "-" + "usr"
	ntGRB := &v3.GlobalRoleBinding{
		ObjectMeta: v1.ObjectMeta{
			Name: name,
		},
		UserName:       user,
		GlobalRoleName: roleName,
	}

	grb, err := nt.rbLister.Get("", ntGRB.Name)
	if err != nil {
		if !strings.Contains(err.Error(), "not found") {
			return nil, err
		}
		return nt.rbClient.Create(ntGRB)
	}
	return grb, nil
}
