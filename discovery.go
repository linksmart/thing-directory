// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package main

import (
	"fmt"
	"log"
	"net"
	"strings"

	"github.com/grandcat/zeroconf"
	"github.com/linksmart/go-sec/auth/obtainer"
	sc "github.com/linksmart/service-catalog/v3/catalog"
	"github.com/linksmart/service-catalog/v3/client"
	"github.com/linksmart/thing-directory/catalog"
)

// register as a DNS-SD Service
func registerDNSSDService(conf *Config) (func(), error) {
	// escape special characters (https://tools.ietf.org/html/rfc6763#section-4.3)
	instance := strings.ReplaceAll(conf.DNSSD.Publish.Instance, ".", "\\.")
	instance = strings.ReplaceAll(conf.DNSSD.Publish.Instance, "\\", "\\\\")

	log.Printf("DNS-SD: registering as \"%s.%s.%s\", subtype: %s",
		instance, catalog.DNSSDServiceType, conf.DNSSD.Publish.Domain, catalog.DNSSDServiceSubtype)

	var ifs []net.Interface

	for _, name := range conf.DNSSD.Publish.Interfaces {
		iface, err := net.InterfaceByName(name)
		if err != nil {
			return nil, fmt.Errorf("error finding interface %s: %s", name, err)
		}
		if (iface.Flags & net.FlagMulticast) > 0 {
			ifs = append(ifs, *iface)
		} else {
			return nil, fmt.Errorf("interface %s does not support multicast", name)
		}
		log.Printf("DNS-SD: will register to interface: %s", name)
	}

	if len(ifs) == 0 {
		log.Println("DNS-SD: publish interfaces not set. Will register to all interfaces with multicast support.")
	}

	sd, err := zeroconf.Register(
		instance,
		catalog.DNSSDServiceType+","+catalog.DNSSDServiceSubtype,
		conf.DNSSD.Publish.Domain,
		conf.HTTP.BindPort,
		[]string{"td=/td", "version=" + Version},
		ifs,
	)
	if err != nil {
		return sd.Shutdown, err
	}

	return sd.Shutdown, nil
}

// register in LinkSmart Service Catalog
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
			URL:      conf.HTTP.PublicEndpoint,
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
