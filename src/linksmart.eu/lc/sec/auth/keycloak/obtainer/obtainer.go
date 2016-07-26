// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package obtainer

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"

	"linksmart.eu/lc/sec/auth/obtainer"
	"log"
	"strconv"
)

const (
	ticketPath = "/protocol/openid-connect/token"
	driverName = "keycloak"
)

type KeycloakObtainer struct{}

var logger *log.Logger

func init() {
	// Initialize the logger
	logger = log.New(os.Stdout, fmt.Sprintf("[%s] ", driverName), 0)
	v, err := strconv.Atoi(os.Getenv("DEBUG"))
	if err == nil && v == 1 {
		logger.SetFlags(log.Ltime | log.Lshortfile)
	}

	// Register the driver as a auth/obtainer
	obtainer.Register(driverName, &KeycloakObtainer{})
}

// Login returns the given credentials to be used by RequestTicket in Openid's implicit flow
func (o *KeycloakObtainer) Login(serverAddr, username, password string) (string, error) {
	return fmt.Sprintf("%s:%s", username, password), nil
}

// RequestTicket requests a ticket
func (o *KeycloakObtainer) RequestTicket(serverAddr, credentials, serviceID string) (string, error) {
	res, err := http.PostForm(serverAddr+ticketPath, url.Values{
		"grant_type": {"password"},
		"client_id":  {serviceID},
		"username":   {strings.Split(credentials, ":")[0]},
		"password":   {strings.Split(credentials, ":")[1]},
	})
	if err != nil {
		return "", fmt.Errorf("%s", err)
	}
	logger.Println("RequestTicket()", res.Status)

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%s", res.Status)
	}

	b, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()
	if err != nil {
		return "", fmt.Errorf("%s", err)
	}

	var body struct {
		Token string `json:"access_token"`
	}
	err = json.Unmarshal(b, &body)
	if err != nil {
		return "", fmt.Errorf("%s", err)
	}

	return body.Token, nil
}

// Logout expires the ticket
func (o *KeycloakObtainer) Logout(serverAddr, TGT string) error {
	return fmt.Errorf("Logout() Not Implemented.")
}
