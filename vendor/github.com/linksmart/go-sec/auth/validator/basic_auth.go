package validator

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/linksmart/go-sec/auth/obtainer"
)

// Cached clients for Basic auth
var (
	clientsMu sync.Mutex
	clients   = make(map[string]*obtainer.Client)
)

const clientExpiration = 10 * time.Minute

// basicAuth obtains a token for the given credentials
//	Tokens are cached by obtainer client and are only renewed if no longer valid
func (v *Validator) basicAuth(credentials string) (string, int, error) {

	clientsMu.Lock()
	client, found := clients[credentials]
	if !found {
		defer clientsMu.Unlock()

		b, err := base64.StdEncoding.DecodeString(credentials)
		if err != nil {
			return "", http.StatusBadRequest, fmt.Errorf("basic auth: invalid encoding: %s", err)
		}

		pair := strings.SplitN(string(b), ":", 2)
		if len(pair) != 2 {
			return "", http.StatusBadRequest, fmt.Errorf("basic auth: invalid format for credentials")
		}

		client, err = obtainer.NewClient(v.driverName, v.serverAddr, pair[0], pair[1], v.clientID)
		if err != nil {
			return "", http.StatusInternalServerError, fmt.Errorf("basic auth: unable to create a client to obtain tokens: %s", err)
		}

		tokenString, errCode, err := v.obtainValidToken(client)
		if err != nil {
			return "", errCode, fmt.Errorf("basic auth: %s", err)
		}

		clients[credentials] = client
		time.AfterFunc(clientExpiration, func() {
			clientsMu.Lock()
			delete(clients, credentials)
			clientsMu.Unlock()
		})

		return tokenString, http.StatusOK, nil
	}
	clientsMu.Unlock()
	return v.obtainValidToken(client)
}

func (v *Validator) obtainValidToken(client *obtainer.Client) (string, int, error) {
	tokenString, err := client.Obtain()
	if err != nil {
		return "", http.StatusUnauthorized, fmt.Errorf("unable to obtain token: %s", err)
	}

	valid, _, err := v.driver.Validate(v.serverAddr, v.clientID, tokenString)
	if err != nil {
		return "", http.StatusInternalServerError, fmt.Errorf("validation error: %s", err)
	}
	if !valid {
		tokenString, err = client.Renew()
		if err != nil {
			return "", http.StatusUnauthorized, fmt.Errorf("unable to renew token: %s", err)
		}
	}
	return tokenString, http.StatusOK, nil
}
