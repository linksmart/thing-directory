// Copyright 2016 Fraunhofer Institute for Applied Information Technology FIT

package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

type router struct {
	*mux.Router
}

func newRouter() *router {
	return &router{mux.NewRouter().StrictSlash(false).SkipClean(true)}
}

func (r *router) get(path string, handler http.Handler) {
	r.Methods("GET").Path(path).Handler(handler)
	r.Methods("GET").Path(fmt.Sprintf("%s/", path)).Handler(handler)
}

func (r *router) post(path string, handler http.Handler) {
	r.Methods("POST").Path(path).Handler(handler)
	r.Methods("POST").Path(fmt.Sprintf("%s/", path)).Handler(handler)
}

func (r *router) put(path string, handler http.Handler) {
	r.Methods("PUT").Path(path).Handler(handler)
	r.Methods("PUT").Path(fmt.Sprintf("%s/", path)).Handler(handler)
}

func (r *router) delete(path string, handler http.Handler) {
	r.Methods("DELETE").Path(path).Handler(handler)
	r.Methods("DELETE").Path(fmt.Sprintf("%s/", path)).Handler(handler)
}

func (r *router) patch(path string, handler http.Handler) {
	r.Methods("PATCH").Path(path).Handler(handler)
	r.Methods("PATCH").Path(fmt.Sprintf("%s/", path)).Handler(handler)
}

func (r *router) head(path string, handler http.Handler) {
	r.Methods("HEAD").Path(path).Handler(handler)
	r.Methods("HEAD").Path(fmt.Sprintf("%s/", path)).Handler(handler)
}

func (r *router) options(path string, handler http.Handler) {
	r.Methods("OPTIONS").Path(path).Handler(handler)
	r.Methods("OPTIONS").Path(fmt.Sprintf("%s/", path)).Handler(handler)
}

func optionsHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func indexHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `<html>`)
	fmt.Fprintf(w, `<h1>LinkSmart Thing Directory</h1>`)
	if Version != "" {
		fmt.Fprintf(w, `<p>Version: %s</p>`, Version)
	}
	fmt.Fprintf(w, `<p><a href="https://github.com/linksmart/thing-directory">https://github.com/linksmart/thing-directory</a></p>`)
	fmt.Fprintf(w, `<p>RESTful directory endpoint: <a href="/td">/td</a></p>`)
	fmt.Fprintf(w, `<p>API Documentation: <a href="https://linksmart.github.io/swagger-ui/dist/?url=https://raw.githubusercontent.com/linksmart/thing-directory/master/apidoc/openapi-spec.yml">Swagger UI</a></p>`)
	fmt.Fprintf(w, `
<p><a href="" id="swagger">Try it out!</a> (experimnental; requires internet connection on both server and client sides)</p>
<script type="text/javascript">
window.onload = function(){
    document.getElementById("swagger").href = "//linksmart.github.io/swagger-ui/dist/?url=" + window.location.toString() + "openapi-spec-proxy" + window.location.pathname;
}
</script>
`)
	fmt.Fprintf(w, `<html>`)
}

func apiSpecProxy(w http.ResponseWriter, req *http.Request) {
	var version = "master"
	if Version != "" {
		version = Version
	}

	// get the spec
	var openapiSpecs = "https://raw.githubusercontent.com/linksmart/thing-directory/" + version + "/apidoc/openapi-spec.yml"
	res, err := http.Get(openapiSpecs)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error querying Open API specs: %s", err)
		return
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		w.WriteHeader(res.StatusCode)
		fmt.Fprintf(w, "GET %s: %s", openapiSpecs, res.Status)
		return
	}

	// write the spec as response
	_, err = io.Copy(w, res.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error responding Open API specs: %s", err)
		return
	}

	// append basename as server URL to api specs
	params := mux.Vars(req)
	if params["basepath"] != "" {
		basePath := strings.TrimSuffix(params["basepath"], "/")
		if !strings.HasPrefix(basePath, "/") {
			basePath = "/" + basePath
		}
		w.Write([]byte("\nservers: [url: " + basePath + "]"))
	}
}
