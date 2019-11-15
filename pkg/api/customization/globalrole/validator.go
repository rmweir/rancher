package globalrole

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"

	"github.com/rancher/norman/types"
	v3 "github.com/rancher/types/apis/management.cattle.io/v3"
)

type Wrapper struct {
	GlobalRoleClient v3.GlobalRoleInterface
}

func (w Wrapper) Validator(request *types.APIContext, schema *types.Schema, data map[string]interface{}) error {
	if request.Method != http.MethodPut {
		return nil
	}

	gr, err := w.GlobalRoleClient.Get(request.ID, v1.GetOptions{})
	if err != nil {
		return err
	}

	if gr.Builtin == true {
		// Drop everything but locked and defaults. If it's builtin nothing else can change.
		for k := range data {
			if k == "newUserDefault" {
				continue
			}
			delete(data, k)
		}

	}
	return nil
}
