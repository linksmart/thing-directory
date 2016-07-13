// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"
)

const (
	// HTTPBackboneName is the fully qualified java class name for HTTP backbone used in LS GC
	HTTPBackboneName = "eu.linksmart.gc.network.backbone.protocol.http.HttpImpl"
	// ProtocolTypeREST used in LC Service registration for Web/HTTP services
	ProtocolTypeREST = "REST"
	// RESTEndpointURL is the key in Protocol.Endpoint describing the endpoint of an Web/HTTP service
	RESTEndpointURL = "url"
	// ServiceAttributeRegistration is the attribute key for TunneledService storing the (modified) Service object
	ServiceAttributeRegistration = "SC"
	// ServiceAttributeID is the attribute key for TunneledService storing the Service ID
	ServiceAttributeID = "SID"
	// ServiceAttributeDescription is the attribute key for TunneledService storing the Service description
	ServiceAttributeDescription = "DESCRIPTION"
)

// GCPublisher is a catalog Listener publishing catalog updates to the GlobalConnect
type GCPublisher struct {
	serviceEndpoint url.URL
	services        map[string]syncedService
	mutex           sync.RWMutex
}

// TunneledService describes a service tunneled with LinkSmart GC
type TunneledService struct {
	Endpoint       string                 `json:"Endpoint"`
	BackboneName   string                 `json:"BackboneName"`
	VirtualAddress string                 `json:"VirtualAddress,omitempty"` // set in responses and used for DELETE
	Attributes     map[string]interface{} `json:"Attributes"`
}

type syncedService struct {
	vad     string    // LSGC Virtual Address
	service Service   // LSLC Service
	lasSync time.Time // timestamp of the last sync
}

// NewTunneledService creates a TunneledService given a Service
func NewTunneledService(s *Service, vad string) (*TunneledService, error) {
	// check if REST protocol is configured
	var restProtocol Protocol
	for _, p := range s.Protocols {
		if p.Type == ProtocolTypeREST {
			restProtocol = p
			break
		}
	}
	if restProtocol.Type == "" {
		return nil, fmt.Errorf("Service without a configured REST protocol. Cannot be tunneled in GC")
	}

	// check if REST protocol has endpoint defined
	endpoint, ok := restProtocol.Endpoint[RESTEndpointURL]
	if !ok {
		return nil, fmt.Errorf("Service with a misconfigured REST protocol (no 'url'). Cannot be tunneled in GC")
	}
	endpointURL, _ := endpoint.(string)

	return &TunneledService{
		Endpoint:     endpointURL,
		BackboneName: HTTPBackboneName,
		Attributes: map[string]interface{}{
			ServiceAttributeID:           s.Id,
			ServiceAttributeDescription:  s.Description,
			ServiceAttributeRegistration: *s,
		},
		VirtualAddress: vad,
	}, nil

}

func tunneledServiceFromResponse(res *http.Response) (*TunneledService, error) {
	decoder := json.NewDecoder(res.Body)
	defer res.Body.Close()

	var ts *TunneledService
	err := decoder.Decode(&ts)
	if err != nil {
		return nil, err
	}
	return ts, nil
}

func (l *GCPublisher) added(s Service) {
	if !s.isGCTunnelable() {
		logger.Printf("Ignoring service that cannot be tunneled in GC: %v\n", s.Id)
		return
	}

	s.Type = ApiRegistrationType
	l.mutex.Lock()
	ssvc, ok := l.services[s.Id]
	// create new service entry if doesn't exist (actually shouldn't)
	if !ok {
		ssvc = syncedService{
			service: s,
		}
	}

	// create tunneled service
	tsvc, err := NewTunneledService(&ssvc.service, "")
	if err != nil {
		logger.Printf("Error creating GC Tunneled Service from the given Service: %v\n", err.Error())
		l.mutex.Unlock()
		return
	}

	// register in the GC
	b, _ := json.Marshal(tsvc)
	res, err := http.Post(l.serviceEndpoint.String(), "application/json", bytes.NewReader(b))

	if err != nil {
		logger.Printf("Error publishing new Service in GC: %v\n", err.Error())
		l.mutex.Unlock()
		return
	}

	// FIXME: should return http.StatusCreated (201)!
	if res.StatusCode != http.StatusOK {
		logger.Printf("Error publishing new Service in GC. Tunneling Service returns: %v\n", res.StatusCode)
		l.mutex.Unlock()
		return
	}

	// Synced successfully
	ssvc.lasSync = time.Now()

	// Parse the response to retrieve VAD
	ts, err := tunneledServiceFromResponse(res)
	if err != nil {
		logger.Printf("Error parsing the Tunneling Service response: %v\n", err.Error())
		l.mutex.Unlock()
		return
	}
	ssvc.vad = ts.VirtualAddress
	logger.Printf("Published service %v in the GC, VAD: %v\n", s.Id, ssvc.vad)

	l.services[s.Id] = ssvc
	l.mutex.Unlock()
}

func (l *GCPublisher) updated(s Service) {
	////////////////////////////////////////////
	// DISABLED FOR THE TIME BEING
	// NM currently assigns a new VAD on update, which effectively has no other effect
	// For now it makes more sense to skip update altogether. See LSGC-146
	////////////////////////////////////////////

	// l.mutex.Lock()
	// // check if service is known
	// ssvc, ok := l.services[s.Id]
	// if !ok {
	// 	logger.Printf("Asked to update unknown service %v **will do nothing**", s.Id)
	// }
	// // replace the service
	// ssvc.service = s

	// // create tunneled service
	// tsvc, err := NewTunneledService(&ssvc.service, ssvc.vad)
	// if err != nil {
	// 	logger.Printf("Error creating GC Tunneled Service from the given Service: %v\n", err.Error())
	// 	l.mutex.Unlock()
	// 	return
	// }

	// // update in the GC
	// b, _ := json.Marshal(tsvc)
	// req, _ := http.NewRequest("PUT", l.serviceEndpoint.String(), bytes.NewReader(b))
	// res, err := http.DefaultClient.Do(req)

	// if err != nil {
	// 	logger.Printf("Error updating Service in GC: %v\n", err.Error())
	// 	l.mutex.Unlock()
	// 	return
	// }

	// if res.StatusCode != http.StatusOK {
	// 	logger.Printf("Error updating Service in GC. Tunneling Service returns: %v\n", res.StatusCode)
	// 	l.mutex.Unlock()
	// 	return
	// }

	// // Synced successfully
	// ssvc.lasSync = time.Now()

	// // Parse the response to retrieve VAD
	// ts, err := tunneledServiceFromResponse(res)
	// if err != nil {
	// 	logger.Printf("Error parsing the Tunneling Service response: %v\n", err.Error())
	// 	l.mutex.Unlock()
	// 	return
	// }
	// ssvc.vad = ts.VirtualAddress
	// logger.Printf("Updated service %v in the GC, VAD: %v\n", s.Id, ssvc.vad)

	// l.services[s.Id] = ssvc
	// l.mutex.Unlock()
}

func (l *GCPublisher) deleted(id string) {
	l.mutex.Lock()
	// check if service is known
	ssvc, ok := l.services[id]

	if !ok {
		logger.Printf("Asked to delete unknown (not tunnellable?) service %v **will do nothing**", id)
		l.mutex.Unlock()
		return
	}

	// delete in the GC
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/%s", l.serviceEndpoint.String(), ssvc.vad), bytes.NewReader([]byte{}))
	res, err := http.DefaultClient.Do(req)

	if err != nil {
		logger.Printf("Error deleting Service in GC: %v\n", err.Error())
		delete(l.services, id)
		l.mutex.Unlock()
		return
	}

	if res.StatusCode != http.StatusOK {
		logger.Printf("Error deleting Service in GC. Tunneling Service returns: %v\n", res.StatusCode)
		delete(l.services, id)
		l.mutex.Unlock()
		return
	}

	logger.Printf("Deleted service %v from the GC, VAD: %v\n", id, ssvc.vad)

	delete(l.services, id)
	l.mutex.Unlock()
}

// NewGCPublisher instantiates a GCPublisher
func NewGCPublisher(endpoint url.URL) *GCPublisher {
	return &GCPublisher{
		serviceEndpoint: endpoint,
		services:        make(map[string]syncedService),
		mutex:           sync.RWMutex{},
	}
}
