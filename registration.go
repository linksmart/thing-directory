// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package main

import (
	"fmt"

	"github.com/linksmart/go-sec/auth/obtainer"
	sc "github.com/linksmart/service-catalog/v3/catalog"
	"github.com/linksmart/service-catalog/v3/client"
)

func registerInServiceCatalog(conf *Config) (func() error, error) {

	cat := conf.ServiceCatalog

	service := sc.Service{
		ID:          conf.ServiceID,
		Type:        "_linksmart-td._tcp",
		Title:       "LinkSmart Thing Directory",
		Description: conf.Description,
		APIs: []sc.API{{
			ID:    "things",
			Title: "Thing Directory API",
			//Description: "API description",
			Protocol: "HTTP",
			URL:      conf.PublicEndpoint,
			Spec: sc.Spec{
				MediaType: "application/vnd.oai.swagger;version=3.0.0",
				URL:       "https://raw.githubusercontent.com/linksmart/thing-directory/master/apidoc/openapi-spec.yml",
				//Schema:    map[string]interface{}{},
			},
			Meta: map[string]interface{}{
				"apiVersion": Version,
			},
		}},
		Doc: "https://github.com/linksmart/thing-directory",
		//Meta: map[string]interface{}{},
		TTL: uint32(conf.ServiceCatalog.Ttl),
	}

	var ticket *obtainer.Client
	var err error
	if cat.Auth.Enabled {
		// Setup ticket client
		ticket, err = obtainer.NewClient(cat.Auth.Provider, cat.Auth.ProviderURL, cat.Auth.Username, cat.Auth.Password, cat.Auth.ClientID)
		if err != nil {
			return nil, fmt.Errorf("error creating auth client: %s", err)
		}
	}

	stopRegistrator, _, err := client.RegisterServiceAndKeepalive(cat.Endpoint, service, ticket)
	if err != nil {
		return nil, fmt.Errorf("error registering service: %s", err)
	}

	return stopRegistrator, nil
}