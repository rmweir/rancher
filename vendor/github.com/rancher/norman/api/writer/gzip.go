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

var gzPool = sync.Pool {
	New: func() interface{} {
		w := gzip.NewWriter(ioutil.Discard)
		return w
	},
}

type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w *gzipResponseWriter) WriteHeader(status int) {
	w.Header().Del("Content-Length")
	w.ResponseWriter.WriteHeader(status)
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
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
	gz := gzPool.Get().(*gzip.Writer)
	defer gzPool.Put(gz)

	gz.Reset(request.Response)
	defer gz.Close()

	request.Response = &gzipResponseWriter{ResponseWriter: request.Response, Writer: gz}
	return nil
}