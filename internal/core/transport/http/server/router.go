package core_http_server

import (
	"fmt"
	"net/http"
)

type ApiVersion string

var (
	ApiVersion1 = ApiVersion("v1")
	ApiVersion2 = ApiVersion("v2")
	ApiVersion3 = ApiVersion("v3")
)


type APIVersionRouter struct{
	*http.ServeMux
	apiVersion ApiVersion
}

func NewAPIVersionRouter(apiVersion ApiVersion) *APIVersionRouter{
	return &APIVersionRouter{
		ServeMux: http.NewServeMux(),
		apiVersion: apiVersion,
	}
}

func (r *APIVersionRouter) RegisterRouters(routes ...Route){
	for _,roroutes := range routes{
		pattern := fmt.Sprintf("%s %s", roroutes.Method, roroutes.Path)
		r.Handle(pattern, roroutes.Handler)
	}
}