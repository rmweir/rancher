package handler

import (
	"github.com/sirupsen/logrus"
	"net/http"
	"strings"

	"github.com/rancher/norman/httperror"
	"github.com/rancher/norman/types"
)

func DeleteHandler(request *types.APIContext, next types.RequestHandler) error {
	store := request.Schema.Store
	if store == nil {
		return httperror.NewAPIError(httperror.NotFound, "no store found")
	}

	obj, err := store.Delete(request, request.Schema, request.ID)
	if err != nil {
		return err
	}
	if strings.Contains(request.Request.Header.Get("Accept-Encoding"), "gzip") {
		logrus.Info("TEST starting gzip")
		logrus.Info("TEST setting encoding to gzip")

		request.Response.Header().Del("Content-Length")
		request.Response.Header().Add("Accept-Charset", "utf-8")
		request.Response.Header().Set("Content-Encoding", "gzip")

	}
	if obj == nil {
		request.WriteResponse(http.StatusNoContent, nil)
	} else {
		request.WriteResponse(http.StatusOK, obj)
	}
	return nil
}
