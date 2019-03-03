// +build !no_edge_provider

package register

import (
	"github.com/virtual-kubelet/virtual-kubelet/providers"
	"github.com/virtual-kubelet/virtual-kubelet/providers/edge"
)

func init() {
	register("edge", initEdge)
}

func initEdge(cfg InitConfig) (providers.Provider, error) {
	return edge.NewEdgeProvider(
		cfg.ConfigPath,
		cfg.NodeName,
		cfg.ResourceManager,
		cfg.OperatingSystem,
		cfg.DaemonPort,
	)
}
