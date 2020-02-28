package catalog

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/url"
	"strings"

	paho "github.com/eclipse/paho.mqtt.golang"
	uuid "github.com/satori/go.uuid"
)

const (
	mqttClientIDPrefix = "SC-"
)

type MQTTConf struct {
	Client            MQTTClientConf   `json:"client"`
	AdditionalClients []MQTTClientConf `json:"additionalClients"`
	CommonRegTopics   []string         `json:"commonRegTopics"`
	CommonWillTopics  []string         `json:"commonWillTopics"`
	TopicPrefix       string           `json:"topicPrefix"`
}

type MQTTClientConf struct {
	BrokerID   string   `json:"brokerID"`
	BrokerURI  string   `json:"brokerURI"`
	RegTopics  []string `json:"regTopics"`
	WillTopics []string `json:"willTopics"`
	QoS        byte     `json:"qos"`
	Username   string   `json:"username,omitempty"`
	Password   string   `json:"password,omitempty"`
	CaFile     string   `json:"caFile,omitempty"`   // trusted CA certificates file path
	CertFile   string   `json:"certFile,omitempty"` // client certificate file path
	KeyFile    string   `json:"keyFile,omitempty"`  // client private key file path
}

func (c MQTTConf) Validate() error {

	for _, client := range append(c.AdditionalClients, c.Client) {
		if client.BrokerURI == "" {
			continue
		}
		_, err := url.Parse(client.BrokerURI)
		if err != nil {
			return err
		}
		if client.QoS > 2 {
			return fmt.Errorf("QoS must be 0, 1, or 2")
		}
		if len(c.CommonRegTopics) == 0 && len(client.RegTopics) == 0 {
			return fmt.Errorf("regTopics not defined")
		}
	}
	return nil
}

func (client MQTTClientConf) pahoOptions() (*paho.ClientOptions, error) {
	opts := paho.NewClientOptions() // uses defaults: https://godoc.org/github.com/eclipse/paho.mqtt.golang#NewClientOptions
	opts.AddBroker(client.BrokerURI)
	opts.SetClientID(fmt.Sprintf("%s%s", mqttClientIDPrefix, uuid.NewV4().String()))

	if client.Username != "" {
		opts.SetUsername(client.Username)
	}
	if client.Password != "" {
		opts.SetPassword(client.Password)
	}

	// TLS CONFIG
	tlsConfig := &tls.Config{}
	if client.CaFile != "" {
		if !strings.HasPrefix(client.BrokerURI, "ssl") {
			logger.Printf("MQTT: Warning: Configuring TLS with a non-SSL protocol: %s", client.BrokerURI)
		}
		// Import trusted certificates from CAfile.pem.
		// Alternatively, manually add CA certificates to
		// default openssl CA bundle.
		tlsConfig.RootCAs = x509.NewCertPool()
		pemCerts, err := ioutil.ReadFile(client.CaFile)
		if err == nil {
			tlsConfig.RootCAs.AppendCertsFromPEM(pemCerts)
		}
	}
	if client.CertFile != "" && client.KeyFile != "" {
		// Import client certificate/key pair
		cert, err := tls.LoadX509KeyPair(client.CertFile, client.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("error loading client keypair: %s", err)
		}
		// Just to print out the client certificate..
		cert.Leaf, err = x509.ParseCertificate(cert.Certificate[0])
		if err != nil {
			return nil, fmt.Errorf("error parsing client certificate: %s", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}
	opts.SetTLSConfig(tlsConfig)

	return opts, nil
}
