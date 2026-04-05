// Package scaffold provides port allocation for Tilt host-side port-forwards.
//
// All containers listen on port 3000 (HTTP) and 9229 (node debug) / 40000 (go debug).
// Host-side port-forwards are incrementally assigned to avoid collisions.
package scaffold

import "github.com/dag7/ironplate/internal/config"

const (
	// BaseForwardPort is the starting host-side HTTP forward port.
	BaseForwardPort = 3010

	// BaseNodeDebugPort is the starting host-side debug forward port for Node.js services.
	BaseNodeDebugPort = 9230

	// BaseGoDebugPort is the starting host-side debug forward port for Go services.
	BaseGoDebugPort = 40001

	// ContainerHTTPPort is the fixed container HTTP port for all services.
	ContainerHTTPPort = 3000

	// ContainerNodeDebugPort is the fixed container debug port for Node.js services.
	ContainerNodeDebugPort = 9229

	// ContainerGoDebugPort is the fixed container debug port for Go services.
	ContainerGoDebugPort = 40000
)

// NextForwardPort returns the next available host-side HTTP forward port
// based on existing services in the config.
func NextForwardPort(existing []config.ServiceSpec) int {
	maxPort := BaseForwardPort - 1
	for _, svc := range existing {
		if svc.Port > maxPort {
			maxPort = svc.Port
		}
	}
	return maxPort + 1
}

// NextDebugForwardPort returns the next available host-side debug forward port
// based on existing services and the new service type.
func NextDebugForwardPort(existing []config.ServiceSpec, serviceType string) int {
	if isGoService(serviceType) {
		return nextGoDebugPort(existing)
	}
	return nextNodeDebugPort(existing)
}

func nextNodeDebugPort(existing []config.ServiceSpec) int {
	maxPort := BaseNodeDebugPort - 1
	for _, svc := range existing {
		if isNodeService(svc.Type) && svc.Port > 0 {
			// Estimate debug port from service index
			maxPort = BaseNodeDebugPort + countNodeServices(existing) - 1
		}
	}
	if maxPort < BaseNodeDebugPort {
		return BaseNodeDebugPort
	}
	return maxPort + 1
}

func nextGoDebugPort(existing []config.ServiceSpec) int {
	goCount := 0
	for _, svc := range existing {
		if isGoService(svc.Type) {
			goCount++
		}
	}
	return BaseGoDebugPort + goCount
}

func countNodeServices(services []config.ServiceSpec) int {
	count := 0
	for _, svc := range services {
		if isNodeService(svc.Type) {
			count++
		}
	}
	return count
}

func isNodeService(serviceType string) bool {
	return serviceType == "node-api" || serviceType == "nextjs"
}

func isGoService(serviceType string) bool {
	return serviceType == "go-api" || serviceType == "go-worker"
}
