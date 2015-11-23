package catalog

import (
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"linksmart.eu/lc/sec/auth/obtainer"
)

// Serves static and all /static/ctx files as ld+json
func NewStaticHandler(staticDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if strings.HasPrefix(req.RequestURI, StaticLocation+"/ctx/") {
			w.Header().Set("Content-Type", "application/ld+json")
		}
		urlParts := strings.Split(req.URL.Path, "/")
		http.ServeFile(w, req, filepath.Join(staticDir, strings.Join(urlParts[2:], "/")))
	}
}

// Constructs and submits an HTTP request and returns the response
func HTTPRequest(method string, url string, headers map[string][]string, body io.Reader,
	ticket *obtainer.Client) (*http.Response, error) {

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	// Set headers
	for key, val := range headers {
		req.Header.Set(key, strings.Join(val, ";"))
	}

	// Do authenticated request if ticket client is provided
	if ticket != nil {
		return HTTPDoAuth(req, ticket)
	}

	// No auth
	return http.DefaultClient.Do(req)
}

// Send an HTTP request with X-Auth-Token entity-header.
//	Ticket is renewed once in case of failure.
func HTTPDoAuth(req *http.Request, ticket *obtainer.Client) (*http.Response, error) {
	X_Auth_Token, err := ticket.Obtain()
	if err != nil {
		return nil, err
	}

	// Set auth header and send the request
	req.Header.Set("X-Auth-Token", X_Auth_Token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, errors.New("HTTPDoAuth() Unexpected empty HTTP response.")
	}

	if res.StatusCode == http.StatusUnauthorized {
		// Get a new ticket and retry again
		logger.Println("HTTPDoAuth() Invalid authentication ticket.")
		X_Auth_Token, err = ticket.Renew()
		if err != nil {
			return nil, err
		}
		logger.Println("HTTPDoAuth() Ticket was renewed.")

		// Reset the header and try again
		req.Header.Set("X-Auth-Token", X_Auth_Token)
		return http.DefaultClient.Do(req)
	}

	return res, nil
}
