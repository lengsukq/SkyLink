package api

import "time"

const (
	serverShutdownTimeout  = 5 * time.Second
	easyTierStartTimeout   = 10 * time.Second
	easyTierRestartTimeout = 15 * time.Second
	easyTierInstallTimeout = 60 * time.Second
	ddnsPublicIPTimeout    = 10 * time.Second
)
