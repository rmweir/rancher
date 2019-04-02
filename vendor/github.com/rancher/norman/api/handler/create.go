package handler

import (
	"github.com/sirupsen/logrus"
	"net/http"
	"strings"

	"github.com/rancher/norman/httperror"
	"github.com/rancher/norman/types"
)

func CreateHandler(apiContext *types.APIContext, next types.RequestHandler) error {
	var err error

	data, err := ParseAndValidateBody(apiContext, true)
	if err != nil {
		return err
	}

	store := apiContext.Schema.Store
	if store == nil {
		return httperror.NewAPIError(httperror.NotFound, "no store found")
	}

	data, err = store.Create(apiContext, apiContext.Schema, data)
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
	apiContext.WriteResponse(http.StatusCreated, data)
	return nil
}
