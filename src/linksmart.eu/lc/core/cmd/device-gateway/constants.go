// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package main

import (
	"time"
)

const (
	// Use to invalidate cache during the requests for agent's data
	AgentResponseCacheTTL time.Duration = time.Duration(3) * time.Second

	// DNS-SD service name (type)
	DNSSDServiceTypeDGW  = "_linksmart-dgw._tcp"
	DNSSDServiceTypeMQTT = "_mqtt._tcp"

	// Static resources URL mounting point
	StaticLocation = "/static"

	// Device Catalog URL mounting point
	CatalogLocation = "/rc"
)
