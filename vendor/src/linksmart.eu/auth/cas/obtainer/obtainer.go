package obtainer

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"

	"linksmart.eu/auth"
	"linksmart.eu/auth/obtainer"
)

const (
	ticketPath = "/v1/tickets/"
)

type Obtainer struct {
	serverAddr string
}

func New(serverAddr string) obtainer.Obtainer {
	// Initialize the logger
	auth.InitLogger(ioutil.Discard, os.Stdout, os.Stdout, os.Stderr, "CAS")

	return &Obtainer{serverAddr}
}

// Request Ticker Granting Ticket (TGT) from CAS Server
func (o *Obtainer) Login(username, password string) (string, error) {
	auth.Log.Println("Getting TGT...")
	res, err := http.PostForm(o.serverAddr+ticketPath, url.Values{
		"username": {username},
		"password": {password},
	})
	if err != nil {
		return "", auth.Error(err)
	}
	auth.Log.Println(res.Status)

	// Check for credentials
	if res.StatusCode != http.StatusCreated {
		return "", auth.Errorf(fmt.Sprintf("Unable to obtain ticket (TGT) for user `%s`.", username))
	}

	locationHeader, err := res.Location()
	if err != nil {
		return "", auth.Error(err)
	}

	return path.Base(locationHeader.Path), nil
}

// Request Service Token from CAS Server
func (o *Obtainer) RequestTicket(TGT, serviceID string) (string, error) {
	auth.Log.Println("Getting Service Ticket...")
	res, err := http.PostForm(o.serverAddr+ticketPath+TGT, url.Values{
		"service": {serviceID},
	})
	if err != nil {
		return "", auth.Error(err)
	}
	auth.Log.Println(res.Status)

	body, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()
	if err != nil {
		return "", auth.Error(err)
	}
	res.Body.Close()

	// Check for TGT errors
	if res.StatusCode != http.StatusOK {
		return "", auth.Errorf(string(body))
	}

	return string(body), nil
}

// Expire the Ticket Granting Ticket
func (o *Obtainer) Logout(TGT string) error {
	auth.Log.Println("Logging out (deleting TGT)...")
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s%s%s", o.serverAddr, ticketPath, TGT), nil)
	if err != nil {
		return auth.Error(err)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return auth.Error(err)
	}
	auth.Log.Println(res.Status)

	// Check for server errors
	if res.StatusCode != http.StatusOK {
		return auth.Errorf(res.Status)
	}

	return nil
}
