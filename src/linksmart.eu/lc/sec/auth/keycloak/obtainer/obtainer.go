// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package obtainer

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"linksmart.eu/lc/sec/auth/obtainer"
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

// Login returns the serialized credentials to be used by RequestTicket
func (o *KeycloakObtainer) Login(_, username, password string) (string, error) {
	credentials := map[string]string{
		"username": username,
		"password": password,
	}
	b, err := json.Marshal(&credentials)
	if err != nil {
		return "", fmt.Errorf("Error serializing credentials: %s", err)
	}
	return string(b), nil
}

// RequestTicket requests a ticket
func (o *KeycloakObtainer) RequestTicket(serverAddr, sCredentials, clientID string) (string, error) {
	// de-serialize credentials
	var credentials map[string]string
	json.Unmarshal([]byte(sCredentials), &credentials)

	res, err := http.PostForm(serverAddr+TicketPath, url.Values{
		"grant_type": {"password"},
		"client_id":  {clientID},
		"username":   {credentials["username"]},
		"password":   {credentials["password"]},
	})
	if err != nil {
		return "", fmt.Errorf("%s", err)
	}
	defer res.Body.Close()
	logger.Println("RequestTicket()", res.Status)

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%s", res.Status)
	}

	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("%s", err)
	}

	var body struct {
		Token string `json:"id_token"`
	}
	err = json.Unmarshal(b, &body)
	if err != nil {
		return "", fmt.Errorf("%s", err)
	}

	return body.Token, nil
}

// Logout expires the ticket (Not applicable in the current flow)
func (o *KeycloakObtainer) Logout(_, _ string) error {
	return nil
}
