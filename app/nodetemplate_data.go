package app

import (
	"github.com/rancher/rancher/pkg/controllers/management/globalnamespacerbac"
	"github.com/rancher/types/apis/management.cattle.io/v3"
	"github.com/rancher/types/config"
	k8srbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
)

func addNodeTemplateGRBs(management *config.ManagementContext) error {
	nodeTemplates, err := management.Management.NodeTemplates("").List(v1.ListOptions{})
	if err != nil {
		return err
	}

	for _, nt := range nodeTemplates.Items {

		user := nt.Annotations[globalnamespacerbac.CreatorIDAnn]

		ownerReference := metav1.OwnerReference{
			APIVersion: globalnamespacerbac.RancherManagementAPIVersion,
			Kind:       "nodetemplates",
			Name:       nt.Name,
			UID:        nt.UID,
		}
		ntRole, err := createRole(management, nt, ownerReference)
		if err != nil {
			return err
		}

		_, err = createGRB(management, user, ntRole.Name)
		if err != nil {
			return err
		}
	}
	return nil
}

func createRole(management *config.ManagementContext, nt v3.NodeTemplate, ownerRef metav1.OwnerReference) (*v3.GlobalRole, error) {
	roleName := "grb-nt-" + nt.Name + "-" + nt.Annotations[globalnamespacerbac.CreatorIDAnn]
	ntRole, err  := management.Management.GlobalRoles("").Get(roleName, metav1.GetOptions{})
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
					ResourceNames: []string{nt.Name},
					Verbs:         []string{"*"},
				},
			},
		}
		return management.Management.GlobalRoles("").Create(newRole)
	}
	return ntRole, nil
}

func createGRB(management *config.ManagementContext, user, roleName string) (*v3.GlobalRoleBinding, error) {
	name := "grb-nt-" + roleName + "-" + "usr"
	ntGRB := &v3.GlobalRoleBinding{
		ObjectMeta: v1.ObjectMeta{
			Name: name,
		},
		UserName:       user,
		GlobalRoleName: roleName,
	}

	grb, err := management.Management.GlobalRoleBindings("").Get(ntGRB.Name, metav1.GetOptions{})
	if err != nil {
		if !strings.Contains(err.Error(), "not found") {
			return nil, err
		}
		return management.Management.GlobalRoleBindings("").Create(ntGRB)
	}
	return grb, nil
}