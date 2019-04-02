package handler

import (
	"github.com/sirupsen/logrus"
	"net/http"
	"strings"

	"github.com/rancher/norman/httperror"
	"github.com/rancher/norman/types"
)

func UpdateHandler(apiContext *types.APIContext, next types.RequestHandler) error {
	data, err := ParseAndValidateBody(apiContext, false)
	if err != nil {
		return err
	}

	store := apiContext.Schema.Store
	if store == nil {
		return httperror.NewAPIError(httperror.NotFound, "no store found")
	}

	data, err = store.Update(apiContext, apiContext.Schema, data, apiContext.ID)
	if err != nil {
		return err
	}
	if strings.Contains(apiContext.Request.Header.Get("Accept-Encoding"), "gzip") {
		logrus.Info("TEST starting gzip")
		logrus.Info("TEST setting encoding to gzip")

		apiContext.Response.Header().Del("Content-Length")
		apiContext.Response.Header().Add("Accept-Charset", "utf-8")
		apiContext.Response.Header().Set("Content-Encoding", "gzip")

	}
	apiContext.WriteResponse(http.StatusOK, data)
	return nil
}
