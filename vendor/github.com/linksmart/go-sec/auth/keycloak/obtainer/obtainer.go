// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

// Package obtainer implements OpenID Connect token obtainment from Keycloak
package obtainer

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/linksmart/go-sec/auth/obtainer"
)

const (
	TicketPath = "/protocol/openid-connect/token"
	DriverName = "keycloak"
)

type KeycloakObtainer struct{}

var logger *log.Logger

func init() {
	// Initialize the logger
	logger = log.New(os.Stdout, fmt.Sprintf("[%s] ", DriverName), 0)
	v, err := strconv.Atoi(os.Getenv("DEBUG"))
	if err == nil && v == 1 {
		logger.SetFlags(log.Ltime | log.Lshortfile)
	}

	// Register the driver as a auth/obtainer
	obtainer.Register(DriverName, &KeycloakObtainer{})
}

type Token struct {
	RefreshToken string `json:"refresh_token"`
	IdToken      string `json:"id_token"`
}

// Login queries and returns the token object
func (o *KeycloakObtainer) Login(serverAddr, username, password, clientID string) (string, error) {

	res, err := http.PostForm(serverAddr+TicketPath, url.Values{
		"grant_type": {"password"},
		"client_id":  {clientID},
		"username":   {username},
		"password":   {password},
		"scope":      {"openid"},
	})
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	logger.Println("Login()", res.Status)

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unable to login with username `%s`: %s", username, string(body))
	}

	var token Token
	err = json.Unmarshal(body, &token)
	if err != nil {
		return "", fmt.Errorf("error getting the token: %s", err)
	}
	if len(strings.Split(token.RefreshToken, ".")) != 3 {
		return "", fmt.Errorf("invalid format for refresh_token")
	}

	serialized, _ := json.Marshal(&token)
	return string(serialized), nil
}

// RequestTicket returns the id_token
//  acquired either from the token object given in the parameter sToken or by requesting it from the server
func (o *KeycloakObtainer) RequestTicket(serverAddr, sToken, clientID string) (string, error) {
	// deserialize the token
	var token Token
	json.Unmarshal([]byte(sToken), &token)

	// decode the id_token acquired on Login()
	decoded, err := base64.RawStdEncoding.DecodeString(strings.Split(token.IdToken, ".")[1])
	if err != nil {
		return "", fmt.Errorf("error decoding the id_token: %s", err)
	}
	var idToken map[string]interface{}
	json.Unmarshal(decoded, &idToken)
	// if id_token is still valid, no need to request a new one
	if int64(idToken["exp"].(float64)) > time.Now().Unix() {
		logger.Println("RequestTicket() Using the newly acquired token.")
		return token.IdToken, nil
	}

	// get a new id_token using the refresh_token
	res, err := http.PostForm(serverAddr+TicketPath, url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {clientID},
		"refresh_token": {token.RefreshToken},
	})
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	logger.Println("RequestTicket()", res.Status)

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("error getting a new token: %s", string(body))
	}

	err = json.Unmarshal(body, &token)
	if err != nil {
		return "", fmt.Errorf("error parsing the new token: %s", err)
	}

	return token.IdToken, nil
}

// Logout expires the ticket (Not applicable in the current flow)
func (o *KeycloakObtainer) Logout(_, _ string) error {
	return nil
}
