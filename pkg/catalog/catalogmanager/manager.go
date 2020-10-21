package catalogmanager

import (
	v32 "github.com/rancher/rancher/pkg/apis/management.cattle.io/v3"
	managementv3 "github.com/rancher/rancher/pkg/generated/norman/management.cattle.io/v3"
)

type CatalogManager interface {
	ValidateChartCompatibility(template *managementv3.CatalogTemplateVersion, clusterName string) error
	ValidateKubeVersion(template *managementv3.CatalogTemplateVersion, clusterName string) error
	ValidateRancherVersion(template *managementv3.CatalogTemplateVersion) error
	LatestAvailableTemplateVersion(template *managementv3.CatalogTemplate, clusterName string) (*v32.TemplateVersionSpec, error)
	GetSystemAppCatalogID(templateVersionID, clusterName string) (string, error)
}
