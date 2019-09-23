package nodetemplate

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/api/meta"
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
	creatorIDAnn = "field.cattle.io/creatorId"
	ctLabel      = "io.cattle.field/clusterTemplateId"
)

type nodeTemplateController struct {
	roleLister v3.GlobalRoleLister
	roleClient v3.GlobalRoleInterface
	rbLister   v3.GlobalRoleBindingLister
	rbClient   v3.GlobalRoleBindingInterface
	mgmtCtx    *config.ManagementContext
}

func Register(ctx context.Context, mgmt *config.ManagementContext) {
	nt := nodeTemplateController{
		roleLister: mgmt.Management.GlobalRoles("").Controller().Lister(),
		roleClient: mgmt.Management.GlobalRoles(""),
		rbLister:	mgmt.Management.GlobalRoleBindings("").Controller().Lister(),
		rbClient:   mgmt.Management.GlobalRoleBindings(""),
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

	if err := globalnamespacerbac.CreateRoleAndRoleBinding(globalnamespacerbac.NodeTemplateResource, nodeTemplate.Name,
		globalnamespacerbac.RancherManagementAPIVersion, creatorID, []string{globalnamespacerbac.RancherManagementAPIVersion},
		nodeTemplate.UID,
		[]v3.Member{}, nt.mgmtCtx); err != nil {
		return nil, err
	}

	// old migration logic
	/*
	user := nodeTemplate.Annotations[globalnamespacerbac.CreatorIDAnn]

	ownerReference := metav1.OwnerReference{
		APIVersion: globalnamespacerbac.RancherManagementAPIVersion,
		Kind:       "nodetemplates",
		Name:       nodeTemplate.Name,
		UID:        nodeTemplate.UID,
	}
	ntRole, err := nt.createRole(nodeTemplate, ownerReference)
	if err != nil {
		return nil, err
	}

	_, err = nt.createGRB(user, ntRole.Name)
	if err != nil {
		return nil, err
	}*/
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
