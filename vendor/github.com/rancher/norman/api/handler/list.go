package handler

import (
	"bytes"
	"encoding/gob"
	"github.com/sirupsen/logrus"
	"net/http"
	"strings"

	"github.com/rancher/norman/httperror"
	"github.com/rancher/norman/parse"
	"github.com/rancher/norman/types"
)
type s int
func ListHandler(request *types.APIContext, next types.RequestHandler) error {
	var (
		err  error
		data interface{}
	)

	store := request.Schema.Store
	if store == nil {
		return httperror.NewAPIError(httperror.NotFound, "no store found")
	}

	if request.ID == "" {
		opts := parse.QueryOptions(request, request.Schema)
		// Save the pagination on the context so it's not reset later
		request.Pagination = opts.Pagination
		data, err = store.List(request, request.Schema, &opts)
	} else if request.Link == "" {
		data, err = store.ByID(request, request.Schema, request.ID)
	} else {
		_, err = store.ByID(request, request.Schema, request.ID)
		if err != nil {
			return err
		}
		return request.Schema.LinkHandler(request, nil)
	}
	if err != nil {
		return err
	}

	if strings.Contains(request.Request.Header.Get("Accept-Encoding"), "gzip") {
		logrus.Info("TEST starting gzip")
		logrus.Info("TEST setting encoding to gzip")

		request.Response.Header().Del("Content-Length")
		request.Response.Header().Set("Content-Encoding", "gzip")
	}
	request.WriteResponse(http.StatusOK, data)
	return nil
}

func GetBytes(key interface{}) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(key)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
