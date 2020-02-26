// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

// Package discovery contains utility functions, which help to implement
// various use-cases of executing some logic as a result of DNS-SD service
// lookup
package discovery

import (
	"github.com/farshidtz/elog"
)

var logger *elog.Logger

func init() {
	logger = elog.New("[discovery] ", &elog.Config{
		DebugPrefix: "[discovery-debug] ",
		DebugTrace:  elog.NoTrace,
	})
}
