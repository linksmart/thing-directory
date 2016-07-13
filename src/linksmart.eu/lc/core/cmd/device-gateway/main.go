// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/oleksandr/bonjour"
	"github.com/syndtr/goleveldb/leveldb/opt"

	utils "linksmart.eu/lc/core/catalog"
	catalog "linksmart.eu/lc/core/catalog/resource"
)

var (
	confPath = flag.String("conf", "conf/device-gateway.json", "Device gateway configuration file path")
)

func main() {
	flag.Parse()

	if *confPath == "" {
		flag.Usage()
		os.Exit(1)
	}

	config, err := loadConfig(*confPath)
	if err != nil {
		logger.Printf("Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Agents' process manager
	agentManager := newAgentManager(config)

	// Configure MQTT if required
	mqttConnector := newMQTTConnector(config, agentManager.DataRequestInbox())
	if mqttConnector != nil {
		agentManager.setPublishingChannel(mqttConnector.dataInbox())
		go mqttConnector.start()
	}

	// Start agents
	go agentManager.start()

	// Expose device's resources via REST (include statics and local catalog)
	restServer, err := newRESTfulAPI(config, agentManager.DataRequestInbox())
	if err != nil {
		logger.Println(err.Error())
		os.Exit(1)
	}

	// Setup Storage backend
	var (
		catalogStorage catalog.CatalogStorage
	)
	// use memory storage if not defined otherwise
	switch config.Storage.Type {
	case "", utils.CatalogBackendMemory:
		catalogStorage = catalog.NewMemoryStorage()
	case utils.CatalogBackendLevelDB:
		tempDir := fmt.Sprintf("%s/lslc/dgw-%d.ldb", strings.Replace(os.TempDir(), "\\", "/", -1), time.Now().UnixNano())
		defer os.RemoveAll(tempDir)

		catalogStorage, err = catalog.NewLevelDBStorage(tempDir, &opt.Options{Compression: opt.NoCompression})
		if err != nil {
			logger.Fatalf("Failed to start LevelDB storage: %v\n", err.Error())
		}
	default:
		logger.Fatalf("Could not create catalog API storage. Unsupported type: %v\n", config.Storage.Type)
	}

	catalogController, err := catalog.NewController(catalogStorage, CatalogLocation)
	if err != nil {
		logger.Printf("Failed to start the controller: %v", err.Error())
		catalogStorage.Close()
		os.Exit(1)
	}

	go restServer.start(catalogController)

	// Parse device configurations
	devices := configureDevices(config)
	// register in local catalog
	err = registerInLocalCatalog(devices, catalogController)
	if err != nil {
		logger.Fatalf("Failed to register in local catalog: %v\n", err.Error())
	}
	// register in remote catalogs
	regChannels, wg := registerInRemoteCatalog(devices, config)

	// Register this gateway as a service via DNS-SD
	var bonjourS *bonjour.Server
	if config.DnssdEnabled {
		restConfig, _ := config.Protocols[ProtocolTypeREST].(RestProtocol)
		bonjourS, err = bonjour.Register(config.Description,
			DNSSDServiceTypeDGW,
			"",
			config.Http.BindPort,
			[]string{fmt.Sprintf("uri=%s", restConfig.Location)},
			nil)
		if err != nil {
			logger.Printf("Failed to register DNS-SD service: %s", err.Error())
		} else {
			logger.Println("Registered service via DNS-SD using type", DNSSDServiceTypeDGW)
		}
	}

	// Ctrl+C handling
	handler := make(chan os.Signal, 1)
	signal.Notify(handler,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	for sig := range handler {
		if sig == os.Interrupt {
			logger.Println("Caught interrupt signal...")
			break
		}
	}

	// Stop bonjour registration
	if bonjourS != nil {
		bonjourS.Shutdown()
		time.Sleep(1e9)
	}

	// Shutdown all
	agentManager.stop()
	if mqttConnector != nil {
		mqttConnector.stop()
	}

	// Shutdown catalog API
	err = catalogStorage.Close()
	if err != nil {
		logger.Println(err.Error())
	}

	// Unregister in the remote catalog(s)
	for _, sigCh := range regChannels {
		// Notify if the routine hasn't returned already
		select {
		case sigCh <- true:
		default:
		}
	}
	wg.Wait()

	logger.Println("Stopped")
	os.Exit(0)
}
