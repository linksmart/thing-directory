// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	catalog "code.linksmart.eu/rc/resource-catalog/catalog"
	sc "code.linksmart.eu/sc/service-catalog/service"
)

const (
	registrationTemplate = `
	{
	  "meta": {
	    "serviceType": "",
	    "apiVersion": ""
	  },
	  "protocols": [
	    {
	      "type": "REST",
	      "endpoint": {
	        "url": ""
	      },
	      "methods": [
	        "GET",
	        "POST"
	      ],
	      "content-types": [
	        "application/ld+json"
	      ]
	    }
	  ],
	  "representation": {
	    "application/ld+json": {}
	  }
	}
	`
	defaultTtl = 120
)

func registrationFromConfig(conf *Config) (*sc.Service, error) {
	c := &sc.ServiceConfig{}

	json.Unmarshal([]byte(registrationTemplate), c)
	c.Name = catalog.ApiName
	publicURL, _ := url.Parse(conf.PublicEndpoint)
	c.Host = strings.Split(publicURL.Host, ":")[0]
	c.Description = conf.Description
	c.Ttl = defaultTtl

	// meta
	c.Meta["serviceType"] = catalog.DNSSDServiceType
	c.Meta["apiVersion"] = catalog.ApiVersion

	// protocols
	// port from the bind port, address from the public address
	c.Protocols[0].Endpoint["url"] = fmt.Sprintf("%v%v", conf.PublicEndpoint, conf.ApiLocation)

	return c.GetService()
}
