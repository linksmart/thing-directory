// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

// Package obtainer implements OpenID Connect token obtainment from Keycloak
package obtainer

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/linksmart/go-sec/auth/obtainer"
)

const (
	TokenEndpoint = "/protocol/openid-connect/token"
	DriverName    = "keycloak"
)

type KeycloakObtainer struct{}

func init() {
	// Register the driver as a auth/obtainer
	obtainer.Register(DriverName, &KeycloakObtainer{})
}

type Token struct {
	RefreshToken string `json:"refresh_token"`
	IdToken      string `json:"id_token"`
}

// ObtainToken requests a token in exchange for user credentials.
// This follows the OAuth 2.0 Resource Owner Password Credentials Grant.
// For this flow, the client in Keycloak must have Direct Grant enabled.
func (o *KeycloakObtainer) ObtainToken(serverAddr, username, password, clientID string) (token interface{}, err error) {

	res, err := http.PostForm(serverAddr+TokenEndpoint, url.Values{
		"grant_type": {"password"},
		"client_id":  {clientID},
		"username":   {username},
		"password":   {password},
		"scope":      {"openid"},
	})
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error getting a token: %s", stringifyError(res.StatusCode, body))
	}

	var keycloakToken Token
	err = json.Unmarshal(body, &keycloakToken)
	if err != nil {
		return nil, fmt.Errorf("error decoding the token: %s", err)
	}

	return keycloakToken, nil
}

// TokenString returns the ID Token part of token object
func (o *KeycloakObtainer) TokenString(token interface{}) (tokenString string, err error) {
	if token, ok := token.(Token); ok {
		return token.IdToken, nil
	}
	return "", fmt.Errorf("invalid input token: assertion error")
}

// RenewToken returns the token
//  acquired either from the token object or by requesting a new one using refresh token
func (o *KeycloakObtainer) RenewToken(serverAddr string, oldToken interface{}, clientID string) (newToken interface{}, err error) {
	token, ok := oldToken.(Token)
	if !ok {
		return nil, fmt.Errorf("invalid input token: assertion error")
	}

	// get a new token using the refresh_token
	res, err := http.PostForm(serverAddr+TokenEndpoint, url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {clientID},
		"refresh_token": {token.RefreshToken},
	})
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error getting a new token: %s", stringifyError(res.StatusCode, body))
	}

	var keycloakToken Token
	err = json.Unmarshal(body, &keycloakToken)
	if err != nil {
		return nil, fmt.Errorf("error decoding the new token: %s", err)
	}

	return keycloakToken, nil
}

// Logout expires the ticket (Not applicable in the current flow)
func (o *KeycloakObtainer) RevokeToken(serverAddr string, token interface{}) error {
	// TODO https://www.keycloak.org/docs/latest/securing_apps/#_token_revocation_endpoint
	return nil
}

func stringifyError(status int, body []byte) string {
	if len(body) == 0 {
		return fmt.Sprintf("%d %s", status, http.StatusText(status))
	}
	var m map[string]interface{}
	err := json.Unmarshal(body, &m)
	if err != nil { // error is not json
		return string(body)
	}
	var str []string
	// error
	if _, ok := m["error"]; ok {
		str = append(str, fmt.Sprint(m["error"]))
		delete(m, "error")
	}
	// error_description
	if _, ok := m["error_description"]; ok {
		str = append(str, fmt.Sprint(m["error_description"]))
		delete(m, "error_description")
	}
	// others
	for k, v := range m {
		str = append(str, fmt.Sprintf("%s (%v)", k, v))
	}
	return strings.Join(str, ": ")
}
