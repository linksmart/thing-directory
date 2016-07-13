// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package main

import (
	"fmt"
	"sync"

	catalog "linksmart.eu/lc/core/catalog/resource"

	_ "linksmart.eu/lc/sec/auth/cas/obtainer"
	"linksmart.eu/lc/sec/auth/obtainer"
)

// Parses config into a slice of configured devices
func configureDevices(config *Config) []catalog.Device {
	devices := make([]catalog.Device, 0, len(config.Devices))
	restConfig, _ := config.Protocols[ProtocolTypeREST].(RestProtocol)
	for _, device := range config.Devices {
		r := new(catalog.Device)
		r.Type = catalog.ApiDeviceType
		r.Ttl = device.Ttl
		r.Name = device.Name
		r.Description = device.Description
		r.Meta = device.Meta
		r.Id = device.Name
		r.Resources = []catalog.Resource{}
		for _, resource := range device.Resources {
			res := new(catalog.Resource)
			res.Type = catalog.ApiResourceType
			res.Name = resource.Name
			res.Meta = resource.Meta
			res.Representation = resource.Representation
			res.Id = fmt.Sprintf("%v/%v", r.Id, res.Name)

			res.Protocols = []catalog.Protocol{}
			for _, proto := range resource.Protocols {
				p := new(catalog.Protocol)
				p.Type = string(proto.Type)
				p.Methods = proto.Methods
				p.ContentTypes = proto.ContentTypes
				p.Endpoint = map[string]interface{}{}
				if proto.Type == ProtocolTypeREST {
					p.Endpoint["url"] = fmt.Sprintf("%s%s/%s/%s",
						config.PublicEndpoint,
						restConfig.Location,
						device.Name,
						resource.Name)
				} else if proto.Type == ProtocolTypeMQTT {
					mqtt, ok := config.Protocols[ProtocolTypeMQTT].(MqttProtocol)
					if ok {
						p.Endpoint["url"] = mqtt.URL
						if proto.PubTopic != "" {
							p.Endpoint["pub_topic"] = proto.PubTopic
						} else {
							p.Endpoint["pub_topic"] = fmt.Sprintf("%s/%v/%v", mqtt.Prefix, device.Name, resource.Name)
						}
						if proto.SubTopic != "" {
							p.Endpoint["sub_topic"] = proto.SubTopic
						}
					}
				}
				res.Protocols = append(res.Protocols, *p)
			}

			r.Resources = append(r.Resources, *res)
		}
		devices = append(devices, *r)
	}
	return devices
}

// Register configured devices from a given configuration using provided controller
func registerInLocalCatalog(devices []catalog.Device, controller catalog.CatalogController) error {
	client := catalog.NewLocalCatalogClient(controller)

	for _, r := range devices {
		r.Ttl = 0
		err := catalog.RegisterDevice(client, &r)
		if err != nil {
			return err
		}
	}
	return nil
}

func registerInRemoteCatalog(devices []catalog.Device, config *Config) ([]chan<- bool, *sync.WaitGroup) {
	regChannels := make([]chan<- bool, 0, len(config.Catalog))
	var wg sync.WaitGroup

	if len(config.Catalog) > 0 {
		logger.Println("Will now register in the configured remote catalogs")

		for _, cat := range config.Catalog {
			var ticket *obtainer.Client
			var err error
			if cat.Auth != nil {
				// Setup ticket client
				ticket, err = obtainer.NewClient(cat.Auth.Provider, cat.Auth.ProviderURL, cat.Auth.Username, cat.Auth.Password, cat.Auth.ServiceID)
				if err != nil {
					logger.Println(err.Error())
					continue
				}
			}

			for _, d := range devices {
				sigCh := make(chan bool)
				wg.Add(1)
				go catalog.RegisterDeviceWithKeepalive(cat.Endpoint, cat.Discover, d, sigCh, &wg, ticket)
				regChannels = append(regChannels, sigCh)
			}
		}
	}

	return regChannels, &wg
}
