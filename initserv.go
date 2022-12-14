package libSwagGo

import (
	routing "github.com/qiangxue/fasthttp-routing"
	"github.com/valyala/fasthttp"
)

type HttpMethod uint8

var router = routing.New()

const (
	GET HttpMethod = iota
	POST
)

func addRoute(method HttpMethod, path string, handler routing.Handler) {
	switch method {
	case GET:
		router.Get(path, handler)
	case POST:
		router.Post(path, handler)
	}
}

func AddRoute(method HttpMethod, path string, handler routing.Handler) {
	addRoute(method, path, handler)
}

func InitServer() {
	panic(fasthttp.ListenAndServe(":8080", router.HandleRequest))
}
