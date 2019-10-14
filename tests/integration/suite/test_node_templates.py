import pytest
import time
import base64
from .common import random_str
from .conftest import wait_for
from rancher import ApiError
from kubernetes.client import CustomObjectsApi, CoreV1Api
from kubernetes.client.rest import ApiException

def test_ssh_stored_as_secret(admin_mc, remove_resource):
    admin_client = admin_mc.client
    ssh_key = "notarealsshkey"
    node_template = admin_client.create_node_template(id="nt-" +
                                                      random_str(),
                                                      amazonec2Config={
                                                          "sshKeyContents": ssh_key
                                                      })
    remove_resource(node_template)

    assert "sshKeyContents" not in node_template.keys()

    k8s_corev1_client = CoreV1Api(admin_mc.k8s_client)
    k8s_dynamic_client = CustomObjectsApi(admin_mc.k8s_client)

    def get_dynamic_nt():
        try:
            node_template_id = node_template.id.split(":")[-1]
            return k8s_dynamic_client.get_namespaced_custom_object("management.cattle.io",
                                                                     "v3",
                                                                     admin_mc.user.id,
                                                                     "nodetemplates",
                                                                     node_template_id)
        except ApiException as e:
            assert e.status == 404
            return False

    dynamic_nt = wait_for(get_dynamic_nt)

    ssh_secret_name = dynamic_nt["amazonec2Config"]["sshKeyContents"]
    ssh_secret =k8s_corev1_client.read_namespaced_secret(ssh_secret_name, "cattle-global-data")
    ssh_secret = base64.b64decode(ssh_secret.data["sshKeyContents"])
    assert ssh_secret == b'notarealsshkey'
