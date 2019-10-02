# import pytest
from kubernetes.client import CustomObjectsApi

def test_node_template_delete(admin_mc, admin_cc, admin_pc, remove_resource):
    """Test deleting a nodeTemplate that is in use by a nodePool.
    The nodeTemplate should not be deleted while in use, after the nodePool is
    removed it should delete.
    """
    admin_client = admin_mc.client
    admin_cc_client = admin_cc.client
    client = admin_pc.client
    k8s = CustomObjectsApi(admin_mc.k8s_client)
    body = {
        "metadata": {
            "name": "t4",
            "annotations": {
                "field.cattle.io/creatorId": admin_mc.user.id
            }
            },
        "kind": "NodeTemplate",
        "apiVersion": "management.cattle.io/v3",
    }
    nt = k8s.create_namespaced_custom_object("management.cattle.io", "v3", admin_mc.user.id, 'nodetemplates', body)
    admin_client.get_node_template(name="nt-" + admin_mc.user.id)
    print(dir(CustomObjectsApi(admin_mc.k8s_client)))
    assert False
    print(dir(client))
    assert False
    #ns=admin_cc_client.create_namespace(name=admin_mc.user.id, clusterId=admin_cc.cluster.id)
    print(admin_mc.user.id)
    nt = admin_client.create_node_template(name="test2", azureConfig={}, namespaceId=admin_mc.user.id, labels={"cattle.io/creator": "norman"})
    print(dir(nt))
    assert False