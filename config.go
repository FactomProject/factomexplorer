// Copyright 2015 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package main

import (
	"code.google.com/p/gcfg"
	"os"
)

type ExplorerConfig struct {
	Explorer struct {
		PortNumber  int
		StaticDir   string
		DatabaseDir string
		UseDatabase bool
	}
}

const defaultConfig = `
; ------------------------------------------------------------------------------
; Explorer Settings
; ------------------------------------------------------------------------------

[explorer]
PortNumber	= 8087
StaticDir	= ""
DatabaseDir	= "/tmp/"
UseDatabase	= true
`

// ReadConfig reads the default factomexplorer.conf file and returns the
// ExplorerConfig object corresponding to the state of the conf file.
func ReadConfig() *ExplorerConfig {
	cfg := new(ExplorerConfig)
	filename := os.Getenv("HOME") + "/.factom/factomexplorer.conf"
	err := gcfg.ReadFileInto(cfg, filename)
	if err != nil {
		gcfg.ReadStringInto(cfg, defaultConfig)
	}
	return cfg
}
