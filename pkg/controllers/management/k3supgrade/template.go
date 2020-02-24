package k3supgrade

const k3sMasterPlan = `---
apiVersion: upgrade.cattle.io/v1
kind: Plan
metadata:
  name: k3s-latest
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
    image: rancher/k3s-upgrade`
