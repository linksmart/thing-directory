package validator

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/kylewolfe/simplexml"
	"linksmart.eu/auth"
	"linksmart.eu/auth/validator"
)

const (
	oauthProfilePath        = "/oauth2.0/profile"
	casProtocolValidatePath = "/p3/serviceValidate"
)

type Validator struct {
	*validator.Conf
}

func New(conf interface{}) (validator.Validator, error) {
	// Initialize the logger
	auth.InitLogger(ioutil.Discard, os.Stdout, os.Stdout, os.Stderr, "CAS")

	authConf, ok := conf.(validator.Conf)
	if !ok {
		return nil, auth.Errorf("Bad config structure.")
	}

	return &Validator{&authConf}, nil
}

// Validate Service Ticket (CAS Protocol)
func (v *Validator) Validate(ticket string) (bool, map[string]string, error) {
	auth.Log.Println("Validating Service Token...")

	bodyMap := make(map[string]string)
	res, err := http.Get(fmt.Sprintf("%s%s?service=%s&ticket=%s",
		v.ServerAddr, casProtocolValidatePath, v.ServiceID, ticket))
	if err != nil {
		auth.Err.Println(err.Error())
		return false, bodyMap, auth.Error(err)
	}
	auth.Log.Println(res.Status)

	// Check for server errors
	if res.StatusCode != http.StatusOK {
		auth.Err.Println(res.Status)
		return false, bodyMap, auth.Errorf(res.Status)
	}

	// User attributes / error message
	body, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()
	if err != nil {
		auth.Err.Println(err.Error())
		return false, bodyMap, auth.Error(err)
	}
	res.Body.Close()

	// Create an xml document from response body
	doc, err := simplexml.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		auth.Err.Println(err.Error())
		return false, bodyMap, auth.Errorf("Unexpected error while validating service token.")
	}
	//fmt.Println(string(body))

	// StatusCode is 200 for all responses (valid, expired, missing)
	// Check if response contains authenticationSuccess tag
	success := doc.Root().Search().ByName("authenticationSuccess").One()
	// There is no authenticationSuccess tag
	// Token is invalid or there are response errors
	if success == nil {
		// Check if response contains authenticationFailure tag
		failure := doc.Root().Search().ByName("authenticationFailure").One()
		if failure == nil {
			auth.Err.Println(err.Error())
			return false, bodyMap, auth.Errorf("Unexpected error while validating service token.")
		}
		// Extract the error message
		errMsg, err := failure.Value()
		if err != nil {
			auth.Err.Println(err.Error())
			return false, bodyMap, auth.Errorf("Unexpected error. No error message.")
		}
		bodyMap["error"] = strings.TrimSpace(errMsg)
		return false, bodyMap, nil
	}

	// Token is valid
	auth.Log.Println("Token was valid.")
	// Extract username
	userTag := doc.Root().Search().ByName("authenticationSuccess").ByName("user").One()
	if userTag == nil {
		auth.Err.Println("Could not find `user` from validation response.")
		return false, bodyMap, auth.Errorf("Could not find `user` from validation response.")
	}
	user, err := userTag.Value()
	if err != nil {
		auth.Err.Println(err.Error())
		return false, bodyMap, auth.Errorf("Could not get value of `user` from validation response.")
	}
	// temporary workaround until CAS bug is fixed
	ldapDescription := strings.Split(user, "-")
	if len(ldapDescription) == 2 {
		bodyMap["user"] = ldapDescription[0]
		bodyMap["group"] = ldapDescription[1]
	} else if len(ldapDescription) == 1 {
		bodyMap["user"] = ldapDescription[0]
		bodyMap["group"] = ""
	} else {
		auth.Err.Println("Unexcpected format for `user` in validation response.")
		return false, bodyMap, auth.Errorf("Unexcpected format for `user` in validation response.")
	}

	// Valid token + attributes
	return true, bodyMap, nil
}

// Validate Service Token (OAUTH)
//func (ca *CasAuth) ValidateServiceToken(serviceToken string) (bool, map[string]interface{}, error) {
//	fmt.Println("CAS: Validating Service Token...")

//	var bodyMap map[string]interface{}
//	res, err := http.Get(fmt.Sprintf("%s%s?access_token=%s", ca.conf.CasServer, oauthProfilePath, serviceToken))
//	if err != nil {
//		return false, bodyMap, fErr(err)
//	}
//	fmt.Println("CAS:", res.Status)

//	// Check for server errors
//	if res.StatusCode != http.StatusOK {
//		return false, bodyMap, fErr(fmt.Errorf(res.Status))
//	}

//	// User attributes / error message
//	body, err := ioutil.ReadAll(res.Body)
//	defer res.Body.Close()
//	if err != nil {
//		return false, bodyMap, fErr(err)
//	}
//	res.Body.Close()

//	if len(body) == 0 { // body is empty due to CAS bug
//		fmt.Println("CAS: Token was valid.")
//		return true, bodyMap, nil
//	}

//	err = json.Unmarshal(body, &bodyMap)
//	if err != nil {
//		return false, bodyMap, fErr(err)
//	}
//	// StatusCode is 200 for all responses (valid, expired, missing)
//	// Check the error message
//	errMsg, ok := bodyMap["error"]
//	if ok {
//		fmt.Println("CAS: Error:", errMsg)
//		return false, bodyMap, nil
//	} else {
//		fmt.Println("CAS: Token was valid.")
//	}

//	// Valid token + attributes
//	return true, bodyMap, nil
//}
