// Copyright 2016 Fraunhofer Institute for Applied Information Technology FIT

package main

import (
	"fmt"
	"html/template"
	"io"
	"log"
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
	version := "master"
	if Version != "" {
		version = Version
	}
	spec := strings.NewReplacer("{version}", version).Replace(Spec)

	swaggerUIRelativeScheme := "//" + SwaggerUISchemeLess
	swaggerUISecure := "https:" + swaggerUIRelativeScheme

	w.Header().Set("Content-Type", "text/html")

	data := struct {
		Logo, Version, SourceRepo, Spec, SwaggerUIRelativeScheme, SwaggerUISecure string
	}{LINKSMART, version, SourceCodeRepo, spec, swaggerUIRelativeScheme, swaggerUISecure}

	tmpl := `
<meta charset="UTF-8">

<pre>{{.Logo}}</pre>
<h1>Thing Directory</h1>
<p>Version: {{.Version}}</p>
<p><a href="{{.SourceRepo}}">{{.SourceRepo}}</a></p>
<p>API Documentation: <a href="{{.SwaggerUISecure}}/?url={{.Spec}}">Swagger UI</a></p>
<p><a href="" id="swagger">Try it out!</a> (experimental; requires internet connection on both server and client sides)</p>
<script type="text/javascript">
	window.onload = function(){
	   document.getElementById("swagger").href = "{{.SwaggerUIRelativeScheme}}/?url=" + window.location.toString() + "openapi-spec-proxy" + window.location.pathname;
	}
</script>`

	t, err := template.New("body").Parse(tmpl)
	if err != nil {
		log.Fatalf("Error parsing template: %s", err)
	}
	err = t.Execute(w, data)
	if err != nil {
		log.Fatalf("Error applying template to response: %s", err)
	}
}

func apiSpecProxy(w http.ResponseWriter, req *http.Request) {
	version := "master"
	if Version != "" {
		version = Version
	}
	spec := strings.NewReplacer("{version}", version).Replace(Spec)

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
