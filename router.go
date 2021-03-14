// Copyright 2016 Fraunhofer Institute for Applied Information Technology FIT

package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
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
	const sourceRepo = "https://github.com/linksmart/thing-directory"
	var version = "master"
	if Version != "" {
		version = Version
	}
	var spec = "https://raw.githubusercontent.com/linksmart/thing-directory/" + version + "/apidoc/openapi-spec.yml"
	var swaggerUIRelativeScheme = "//linksmart.github.io/swagger-ui/dist"
	var swaggerUISecure = "https:" + swaggerUIRelativeScheme
	// TODO: check on startup
	_, err := url.ParseRequestURI(swaggerUISecure)
	if err != nil {
		log.Printf("ERROR: Invalid SwaggerUI URL: %s", err)
	}

	w.Header().Set("Content-Type", "text/html")
	var body string
	body += `<html>`
	body += `<h1>LinkSmart Thing Directory</h1>`
	body += fmt.Sprintf(`Version: %s</p>`, version)
	// Source code
	body += fmt.Sprintf(`<p><a href="%s">%s</a></p>`, sourceRepo, sourceRepo)
	// Registration endpoint
	body += `<p>RESTful registration endpoint: <a href="./td">/td</a></p>`
	// Swagger UI
	body += fmt.Sprintf(`<p>API Documentation: <a href="%s/?url=%s">Swagger UI</a></p>`, swaggerUISecure, spec)
	// Interactive Swagger UI
	body += fmt.Sprintf(`<p><a href="" id="swagger">Try it out!</a> (experimental; requires internet connection on both server and client sides)</p>
<script type="text/javascript">
window.onload = function(){
    document.getElementById("swagger").href = "%s/?url=" + window.location.toString() + "openapi-spec-proxy" + window.location.pathname;
}
</script>`, swaggerUIRelativeScheme)
	body += `<html>`

	_, err = w.Write([]byte(body))
	if err != nil {
		log.Printf("ERROR writing HTTP response: %s", err)
	}
}

func apiSpecProxy(w http.ResponseWriter, req *http.Request) {
	var version = "master"
	if Version != "" {
		version = Version
	}
	var spec = "https://raw.githubusercontent.com/linksmart/thing-directory/" + version + "/apidoc/openapi-spec.yml"

	// get the spec
	res, err := http.Get(spec)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, err := fmt.Fprintf(w, "Error querying Open API specs: %s", err)
		if err != nil {
			log.Printf("ERROR writing HTTP response: %s", err)
		}
		return
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		w.WriteHeader(res.StatusCode)
		_, err := fmt.Fprintf(w, "GET %s: %s", spec, res.Status)
		if err != nil {
			log.Printf("ERROR writing HTTP response: %s", err)
		}
		return
	}

	// write the spec as response
	_, err = io.Copy(w, res.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, err := fmt.Fprintf(w, "Error responding Open API specs: %s", err)
		if err != nil {
			log.Printf("ERROR writing HTTP response: %s", err)
		}
		return
	}

	// append basename as server URL to api specs
	params := mux.Vars(req)
	if params["basepath"] != "" {
		basePath := strings.TrimSuffix(params["basepath"], "/")
		if !strings.HasPrefix(basePath, "/") {
			basePath = "/" + basePath
		}
		_, err := w.Write([]byte("\nservers: [url: " + basePath + "]"))
		if err != nil {
			log.Printf("ERROR writing HTTP response: %s", err)
		}
	}
}
