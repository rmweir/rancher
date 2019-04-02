package writer

import (
	"github.com/rancher/norman/types"
	"github.com/sirupsen/logrus"
	"net/http"
	"compress/gzip"
	"io/ioutil"
	"strings"
	"sync"
	"io"
)

var GzPool = sync.Pool {
	New: func() interface{} {
		w := gzip.NewWriter(ioutil.Discard)
		return w
	},
}

type GzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w *GzipResponseWriter) WriteHeader(status int) {
	w.Header().Del("Content-Length")
	w.ResponseWriter.WriteHeader(status)
}

func (w *GzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func Gzip(request *types.APIContext, next types.RequestHandler) error {
	logrus.Info("TEST starting gzip")
	if !strings.Contains(request.Request.Header.Get("Accept-Encoding"), "gzip") {
		next(request, nil)
		return nil
	}
	logrus.Info("TEST setting encoding to gzip")
	request.Request.Header.Set("Content-Encoding", "gzip")
	gz := GzPool.Get().(*gzip.Writer)
	defer GzPool.Put(gz)

	gz.Reset(request.Response)
	// defer gz.Close()

	request.Response = &GzipResponseWriter{ResponseWriter: request.Response, Writer: gz}
	request.Response.WriteHeader(http.StatusOK)
	//request.Response.Write(data)
	return nil
}