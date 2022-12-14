package libSwagGo

// Package fasthttpadaptor provides helper functions for converting net/http
// request handlers to fasthttp-routing handlers.

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/url"

	routing "github.com/qiangxue/fasthttp-routing"
	"github.com/valyala/fasthttp"
)

// NewFastHTTPHandlerFunc wraps net/http handler func to fasthttp
// request handler, so it can be passed to fasthttp server.
//
// While this function may be used for easy switching from net/http to fasthttp,
// it has the following drawbacks comparing to using manually written fasthttp
// request handler:
//
//   - A lot of useful functionality provided by fasthttp is missing
//     from net/http handler.
//   - net/http -> fasthttp handler conversion has some overhead,
//     so the returned handler will be always slower than manually written
//     fasthttp handler.
//
// So it is advisable using this function only for quick net/http -> fasthttp
// switching. Then manually convert net/http handlers to fasthttp handlers
// according to https://github.com/valyala/fasthttp#switching-from-nethttp-to-fasthttp .
func NewFastHTTPHandlerFunc(h http.HandlerFunc) routing.Handler {
	return NewFastHTTPHandler(h)
}

// NewFastHTTPHandler wraps net/http handler to fasthttp request handler,
// so it can be passed to fasthttp server.
//
// While this function may be used for easy switching from net/http to fasthttp,
// it has the following drawbacks comparing to using manually written fasthttp
// request handler:
//
//   - A lot of useful functionality provided by fasthttp is missing
//     from net/http handler.
//   - net/http -> fasthttp handler conversion has some overhead,
//     so the returned handler will be always slower than manually written
//     fasthttp handler.
//
// So it is advisable using this function only for quick net/http -> fasthttp
// switching. Then manually convert net/http handlers to fasthttp handlers
// according to https://github.com/valyala/fasthttp#switching-from-nethttp-to-fasthttp .
func NewFastHTTPHandler(h http.Handler) routing.Handler {
	return func(ctx *routing.Context) error {
		var r http.Request
		if err := ConvertRequest(ctx, &r, true); err != nil {
			ctx.Logger().Printf("cannot parse requestURI %q: %s", r.RequestURI, err)
			ctx.Error("Internal Server Error", fasthttp.StatusInternalServerError)
			return err
		}

		var w netHTTPResponseWriter
		h.ServeHTTP(&w, r.WithContext(ctx))

		ctx.SetStatusCode(w.StatusCode())
		haveContentType := false
		for k, vv := range w.Header() {
			if k == fasthttp.HeaderContentType {
				haveContentType = true
			}

			for _, v := range vv {
				ctx.Response.Header.Add(k, v)
			}
		}
		if !haveContentType {
			// From net/http.ResponseWriter.Write:
			// If the Header does not contain a Content-Type line, Write adds a Content-Type set
			// to the result of passing the initial 512 bytes of written data to DetectContentType.
			l := 512
			if len(w.body) < 512 {
				l = len(w.body)
			}
			ctx.Response.Header.Set(fasthttp.HeaderContentType, http.DetectContentType(w.body[:l]))
		}
		ctx.Write(w.body) //nolint:errcheck
		return nil
	}
}

type netHTTPResponseWriter struct {
	statusCode int
	h          http.Header
	body       []byte
}

func (w *netHTTPResponseWriter) StatusCode() int {
	if w.statusCode == 0 {
		return http.StatusOK
	}
	return w.statusCode
}

func (w *netHTTPResponseWriter) Header() http.Header {
	if w.h == nil {
		w.h = make(http.Header)
	}
	return w.h
}

func (w *netHTTPResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
}

func (w *netHTTPResponseWriter) Write(p []byte) (int, error) {
	w.body = append(w.body, p...)
	return len(p), nil
}

// ConvertRequest convert a fasthttp.Request to an http.Request
// forServer should be set to true when the http.Request is going to passed to a http.Handler.
func ConvertRequest(ctx *routing.Context, r *http.Request, forServer bool) error {
	body := ctx.PostBody()
	strRequestURI := string(ctx.RequestURI())

	rURL, err := url.ParseRequestURI(strRequestURI)
	if err != nil {
		return err
	}

	r.Method = string(ctx.Method())
	r.Proto = "HTTP/1.1"
	r.ProtoMajor = 1
	r.ProtoMinor = 1
	r.ContentLength = int64(len(body))
	r.RemoteAddr = ctx.RemoteAddr().String()
	r.Host = string(ctx.Host())
	r.TLS = ctx.TLSConnectionState()
	r.Body = ioutil.NopCloser(bytes.NewReader(body))
	r.URL = rURL

	if forServer {
		r.RequestURI = strRequestURI
	}

	if r.Header == nil {
		r.Header = make(http.Header)
	} else if len(r.Header) > 0 {
		for k := range r.Header {
			delete(r.Header, k)
		}
	}

	ctx.Request.Header.VisitAll(func(k, v []byte) {
		sk := string(k)
		sv := string(v)

		switch sk {
		case "Transfer-Encoding":
			r.TransferEncoding = append(r.TransferEncoding, sv)
		default:
			r.Header.Set(sk, sv)
		}
	})

	return nil
}
