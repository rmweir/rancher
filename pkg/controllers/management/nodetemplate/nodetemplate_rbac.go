package nodetemplate

import (
	"context"
	"fmt"
	"github.com/rancher/rancher/pkg/namespace"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/labels"
	"strings"

	"github.com/rancher/rancher/pkg/controllers/management/globalnamespacerbac"
	v3 "github.com/rancher/types/apis/management.cattle.io/v3"
	"github.com/rancher/types/config"
	k8srbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/runtime"
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
		rbLister:	mgmt.Management.GlobalRoleBindings("").Controller().Lister(),
		rbClient:   mgmt.Management.GlobalRoleBindings(""),
		ntClient:   mgmt.Management.NodeTemplates(""),
		ntLister:   mgmt.Management.NodeTemplates("").Controller().Lister(),
		npLister:   mgmt.Management.NodePools("").Controller().Lister(),
		npClient:   mgmt.Management.NodePools(""),
		mgmtCtx:    mgmt,
	}

	mgmt.Management.NodeTemplates("").Controller().AddHandler(ctx,"nt-grb-handler", nt.sync)
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
			migratedNTName := fmt.Sprintf("nt-%s-%s", nodeTemplate.Namespace, nodeTemplate.Name)

			globalNodeTemplate, err := nt.ntLister.Get("cattle-global-data", migratedNTName)
			if err != nil {
				// legacy template has not been created yet, create it
				if !strings.Contains(err.Error(), "not found") {
					return nil, err
				}

				globalNodeTemplate = nodeTemplate.DeepCopy()
				globalNodeTemplate.ObjectMeta = metav1.ObjectMeta{
					Name: migratedNTName,
					Namespace: namespace.GlobalNamespace,
					Annotations: nodeTemplate.Annotations,
					Labels: map[string]string{"parentNodeTemplate": string(nodeTemplate.UID)},
				}

				globalNodeTemplate, err = nt.ntClient.Create(globalNodeTemplate)
				if err != nil {
					return nil, err
				}
			}

			npList, err := nt.npLister.List("", labels.Everything())
			if err != nil {
				return nil, err
			}
			for _, np := range npList {
				if np.Spec.NodeTemplateName == fmt.Sprintf("%s:%s", nodeTemplate.Namespace, nodeTemplate.Name) {
					npCopy := np.DeepCopy()
					npCopy.Spec.NodeTemplateName = globalNodeTemplate.Name

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
					nodeCopy.Spec.NodeTemplateName = globalNodeTemplate.Name

					_, err := nt.mgmtCtx.Management.Nodes("").Create(nodeCopy)
					if err != nil {
						return nil, err
					}
				}
			}




			nodeTemplate.Annotations["migratedToGlobal"] = "true"
			_, err = nt.ntClient.Update(nodeTemplate)
			if err != nil {
				return nil, err
			}

			nodeTemplate = globalNodeTemplate
		}
	}

	// Create Role and RBs
	if err := globalnamespacerbac.CreateRoleAndRoleBinding(globalnamespacerbac.NodeTemplateResource, nodeTemplate.Name,
		globalnamespacerbac.RancherManagementAPIVersion, creatorID, []string{globalnamespacerbac.RancherManagementAPIVersion},
		nodeTemplate.UID,
		[]v3.Member{}, nt.mgmtCtx); err != nil {
		return nil, err
	}

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
