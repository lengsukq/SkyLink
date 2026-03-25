package api

import (
	"testing"

	"github.com/skylink/skylink/internal/store"
)

func TestBuildEasyTierReleasePortList(t *testing.T) {
	ports, err := buildEasyTierReleasePortList(store.EasyTierConfig{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ports) == 0 {
		t.Fatalf("expected default ports")
	}

	ports, err = buildEasyTierReleasePortList(store.EasyTierConfig{RPCPortal: "127.0.0.1:19999"})
	if err != nil {
		t.Fatalf("unexpected error with rpc portal: %v", err)
	}
	found := false
	for _, p := range ports {
		if p == 19999 {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected rpc_portal port in allowed list")
	}
}

func TestBuildEasyTierReleasePortList_InvalidRPCPortal(t *testing.T) {
	if _, err := buildEasyTierReleasePortList(store.EasyTierConfig{RPCPortal: "invalid"}); err == nil {
		t.Fatalf("expected invalid rpc_portal error")
	}
}
