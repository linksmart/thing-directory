// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

// Package client offers utility functions for service registration
package client

import (
	"github.com/farshidtz/elog"
)

var logger *elog.Logger

func init() {
	logger = elog.New("[sc] ", &elog.Config{
		DebugPrefix: "[sc-debug] ",
		DebugTrace:  elog.NoTrace,
	})
}
