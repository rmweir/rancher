package k3supgrade

import (
	"github.com/rancher/system-upgrade-controller/pkg/apis/upgrade.cattle.io"
	planv1 "github.com/rancher/system-upgrade-controller/pkg/apis/upgrade.cattle.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubectl/pkg/describe"
)

const k3sMasterPlan = `---
apiVersion: upgrade.cattle.io/v1
kind: Plan
metadata:
  name: k3s-master-plan 
  namespace: system-upgrade
spec:
  concurrency: 1
  version: v1.17.2-k3s1
  nodeSelector:
    matchExpressions:
      - {key: k3s-upgrade, operator: Exists}
  serviceAccountName: default
  drain:
    force: true
  upgrade:
    image: rancher/k3s-upgrade:latest`

// https://github.com/rancher/k3s-upgrade/pull/9
const k3sWorkerPlan = `---
apiVersion: upgrade.cattle.io/v1
kind: Plan
metadata:
  name: k3s-worker-plan
  namespace: system-upgrade
spec:
  concurrency: 1
  version: v1.17.2-k3s1
  # The prepare init container is run before cordon/drain which is run before the upgrade container.
  # Shares the same format as the "upgrade" container
  prepare:
     image: rancher/k3s-upgrade:latest
     args: ["prepare","k3s-master-plan"]
  nodeSelector:
    matchExpressions:
    - {key: k3s-worker-upgrade, operator: Exists}
  serviceAccountName: system-upgrade
  drain:
    force: true
  upgrade:
    image: rancher/k3s-upgrade`

const k3sMasterPlanName = "k3s-master-plan"
const k3sWorkerPlanName = "k3s-worker-plan"
const systemUpgradeServiceAccount = "system-upgrade"
const upgradeImage = "rancher/k3s-upgrade:latest"

var genericPlan = planv1.Plan{
	TypeMeta: metav1.TypeMeta{
		Kind:       "Plan",
		APIVersion: upgrade.GroupName + `/v1`,
	},
	ObjectMeta: metav1.ObjectMeta{
		Namespace: systemUpgradeNS,
		Labels:    map[string]string{rancherManagedPlan: "true"},
	},
	Spec: planv1.PlanSpec{
		Concurrency:        0,
		ServiceAccountName: systemUpgradeServiceAccount,
		Channel:            "",
		Version:            "",
		Secrets:            nil,
		Prepare:            nil,
		Cordon:             false,
		Drain: &planv1.DrainSpec{
			Force: true,
		},
		Upgrade: &planv1.ContainerSpec{
			Image: upgradeImage,
		},
	},
	Status: planv1.PlanStatus{},
}

func generateMasterPlan(version string, concurrency int) (planv1.Plan, error) {
	masterPlan := genericPlan
	masterPlan.Name = k3sMasterPlanName
	masterPlan.Spec.Version = version
	masterPlan.Spec.Concurrency = int64(concurrency)
	// only select master nodes

	masterPlan.Spec.NodeSelector = &metav1.LabelSelector{
		MatchExpressions: []metav1.LabelSelectorRequirement{{

			Key:      describe.LabelNodeRolePrefix + "master",
			Operator: metav1.LabelSelectorOpIn,
			Values:   []string{"true"},
		}},
	}

	return masterPlan, nil
}

func generateWorkerPlan(version string, concurrency int) (planv1.Plan, error) {
	workerPlan := genericPlan
	workerPlan.Name = k3sWorkerPlanName
	workerPlan.Spec.Version = version
	workerPlan.Spec.Concurrency = int64(concurrency)

	// worker plans wait for master plans to complete
	workerPlan.Spec.Prepare = &planv1.ContainerSpec{
		Image:   upgradeImage,
		Command: []string{"prepare", k3sMasterPlanName},
		Args:    nil,
	}

	return workerPlan, nil
}
