package devtools

import (
	"fmt"
	"sort"
	"strings"
)

// Pod represents a Kubernetes pod.
type Pod struct {
	Name      string
	Namespace string
	Status    string
	Ready     string
	Restarts  int
	Age       string
	Node      string
}

// ServiceGroup groups pods by their parent service.
type ServiceGroup struct {
	Name string
	Pods []Pod
}

// ListPods returns all pods in the given namespace.
func ListPods(namespace string) ([]Pod, error) {
	result, err := Kubectl("get", "pods", "-n", namespace, "--no-headers",
		"-o", "custom-columns=NAME:.metadata.name,STATUS:.status.phase,READY:.status.conditions[?(@.type=='Ready')].status,RESTARTS:.status.containerStatuses[0].restartCount,AGE:.metadata.creationTimestamp,NODE:.spec.nodeName")
	if err != nil {
		return nil, err
	}

	var pods []Pod
	for _, line := range strings.Split(strings.TrimSpace(result.Stdout), "\n") {
		if line == "" || line == "<none>" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		restarts := 0
		fmt.Sscanf(fields[3], "%d", &restarts) //nolint:errcheck // best-effort parse, defaults to 0

		pod := Pod{
			Name:      fields[0],
			Namespace: namespace,
			Status:    fields[1],
			Ready:     fields[2],
			Restarts:  restarts,
		}
		if len(fields) >= 5 {
			pod.Age = fields[4]
		}
		if len(fields) >= 6 {
			pod.Node = fields[5]
		}
		pods = append(pods, pod)
	}
	return pods, nil
}

// GroupPodsByService groups pods by their inferred service name.
func GroupPodsByService(pods []Pod) []ServiceGroup {
	groups := make(map[string][]Pod)
	for _, p := range pods {
		service := inferServiceName(p.Name)
		groups[service] = append(groups[service], p)
	}

	var result []ServiceGroup
	for name, pods := range groups {
		result = append(result, ServiceGroup{Name: name, Pods: pods})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}

// PodLogs streams logs from a pod.
func PodLogs(podName, namespace string, follow bool, tail int, container string) error {
	args := []string{"logs", podName, "-n", namespace}
	if follow {
		args = append(args, "-f")
	}
	if tail > 0 {
		args = append(args, fmt.Sprintf("--tail=%d", tail))
	}
	if container != "" {
		args = append(args, "-c", container)
	}
	return KubectlInteractive(args...)
}

// PodExec opens a shell in a pod.
func PodExec(podName, namespace string, container string) error {
	args := []string{"exec", "-it", podName, "-n", namespace}
	if container != "" {
		args = append(args, "-c", container)
	}
	args = append(args, "--", "/bin/sh")
	return KubectlInteractive(args...)
}

// PodPortForward forwards a local port to a pod port.
func PodPortForward(podName, namespace string, localPort, remotePort int) error {
	args := []string{"port-forward", podName, "-n", namespace, fmt.Sprintf("%d:%d", localPort, remotePort)}
	return KubectlInteractive(args...)
}

// DescribePod returns the description of a pod.
func DescribePod(podName, namespace string) error {
	return KubectlInteractive("describe", "pod", podName, "-n", namespace)
}

// StatusIcon returns an emoji for pod status.
func StatusIcon(status string) string {
	switch strings.ToLower(status) {
	case "running":
		return "●" // green when styled
	case "pending":
		return "◐"
	case "succeeded", "completed":
		return "✓"
	case "failed", "error":
		return "✗"
	case "crashloopbackoff":
		return "↻"
	default:
		return "○"
	}
}

// inferServiceName extracts the service name from a pod name
// by removing the trailing replicaset hash and pod hash.
func inferServiceName(podName string) string {
	parts := strings.Split(podName, "-")
	if len(parts) <= 2 {
		return podName
	}
	// Pod names are typically: service-name-replicaset-hash
	// We want to strip the last 2 segments (replicaset + pod hash)
	return strings.Join(parts[:len(parts)-2], "-")
}
