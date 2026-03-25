package api

import (
	"github.com/skylink/skylink/internal/easytier"
	"github.com/skylink/skylink/internal/store"
)

func buildEasyTierReleasePortList(cfg store.EasyTierConfig) ([]int, error) {
	ports := make([]int, 0, len(easytier.DefaultEasyTierPorts)+1)
	portSet := make(map[int]bool)
	for _, p := range easytier.DefaultEasyTierPorts {
		if !portSet[p] {
			portSet[p] = true
			ports = append(ports, p)
		}
	}
	if cfg.RPCPortal == "" {
		return ports, nil
	}
	rpcPort := easytier.ParsePortFromAddress(cfg.RPCPortal)
	if rpcPort == 0 {
		return nil, errf("invalid rpc_portal")
	}
	if !portSet[rpcPort] {
		ports = append(ports, rpcPort)
	}
	return ports, nil
}
