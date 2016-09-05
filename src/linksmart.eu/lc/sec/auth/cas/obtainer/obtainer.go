// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package obtainer

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"

	"linksmart.eu/lc/sec/auth/obtainer"
)

const (
	ticketPath = "/v1/tickets/"
	driverName = "cas"
)

type CASObtainer struct{}

var logger *log.Logger

func init() {
	// Initialize the logger
	logger = log.New(os.Stdout, fmt.Sprintf("[%s] ", driverName), 0)
	v, err := strconv.Atoi(os.Getenv("DEBUG"))
	if err == nil && v == 1 {
		logger.SetFlags(log.Ltime | log.Lshortfile)
	}

	// Register the driver as a auth/obtainer
	obtainer.Register(driverName, &CASObtainer{})
}

// Request Ticker Granting Ticket (TGT) from CAS Server
func (o *CASObtainer) Login(serverAddr, username, password, _ string) (string, error) {
	res, err := http.PostForm(serverAddr+ticketPath, url.Values{
		"username": {username},
		"password": {password},
	})
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	logger.Println("Login()", res.Status)

	// Check for credentials
	if res.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("Unable to obtain ticket (TGT) for user `%s`.", username)
	}

	locationHeader, err := res.Location()
	if err != nil {
		return "", err
	}

	return path.Base(locationHeader.Path), nil
}

// Request Service Token from CAS Server
func (o *CASObtainer) RequestTicket(serverAddr, TGT, serviceID string) (string, error) {
	res, err := http.PostForm(serverAddr+ticketPath+TGT, url.Values{
		"service": {serviceID},
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

	// Check for TGT errors
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%s", string(body))
	}

	return string(body), nil
}

// Expire the Ticket Granting Ticket
func (o *CASObtainer) Logout(serverAddr, TGT string) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s%s%s", serverAddr, ticketPath, TGT), nil)
	if err != nil {
		return err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	logger.Println("Logout()", res.Status)

	// Check for server errors
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf(res.Status)
	}

	return nil
}
