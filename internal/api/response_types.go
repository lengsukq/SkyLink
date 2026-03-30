package api

import (
	"github.com/skylink/skylink/internal/cloudflare"
	"github.com/skylink/skylink/internal/store"
)

type errorResponse struct {
	Error string `json:"error"`
}

type okResponse struct {
	OK bool `json:"ok"`
}

type warningOKResponse struct {
	OK      bool   `json:"ok"`
	Warning string `json:"warning,omitempty"`
}

type mappingsListResponse struct {
	List []store.Mapping `json:"list"`
}

type cfZonesResponse struct {
	Zones []cloudflare.Zone `json:"zones"`
}

type cfDNSRecordsResponse struct {
	Records []cloudflare.DNSRecord `json:"records"`
}

type easyTierCLIOutputResponse struct {
	OK     bool   `json:"ok"`
	Target string `json:"target,omitempty"`
	Stdout string `json:"stdout,omitempty"`
	Stderr string `json:"stderr,omitempty"`
	Error  string `json:"error,omitempty"`
	Hint   string `json:"hint,omitempty"`
}
