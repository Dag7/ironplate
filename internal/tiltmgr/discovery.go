package tiltmgr

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// DiscoveredService represents a service found by parsing the Tiltfile.
type DiscoveredService struct {
	Name  string
	Group string
}

// DiscoveredResources contains all resources parsed from a Tiltfile.
type DiscoveredResources struct {
	Services []DiscoveredService
	Infra    []string
}

var (
	k8sResourcePattern   = regexp.MustCompile(`k8s_resource\(\s*['"]([^'"]+)['"]`)
	dockerBuildPattern   = regexp.MustCompile(`docker_build\(\s*['"][^/'"]*\/([^'"]+)['"]`)
	infraLoadPattern     = regexp.MustCompile(`(?:load|include)\(\s*['"]\.\/k8s\/(?:helm|deployment)\/infra\/([^/'"]+)`)
	localResourcePattern = regexp.MustCompile(`local_resource\(\s*\n?\s*['"]([^'"]+)['"]`)
	loadTiltfilePattern  = regexp.MustCompile(`load\(\s*['"]([^'"]+Tiltfile)['"]`)
	helmPathPattern      = regexp.MustCompile(`k8s/helm/(?:apps|services)/([^/'"]+)`)
	labelsPattern        = regexp.MustCompile(`labels\s*=\s*\[['"]([^'"]+)['"]`)
)

// ParseTiltfile discovers services and infrastructure from a Tiltfile.
func ParseTiltfile(tiltfilePath string) (*DiscoveredResources, error) {
	content, err := os.ReadFile(tiltfilePath)
	if err != nil {
		return nil, err
	}

	tiltfileDir := filepath.Dir(tiltfilePath)
	text := string(content)

	serviceSet := make(map[string]bool)
	infraSet := make(map[string]bool)

	// Extract k8s_resource declarations
	for _, match := range k8sResourcePattern.FindAllStringSubmatch(text, -1) {
		serviceSet[match[1]] = true
	}

	// Extract docker_build image names
	for _, match := range dockerBuildPattern.FindAllStringSubmatch(text, -1) {
		name := match[1]
		if !serviceSet[name] {
			serviceSet[name] = true
		}
	}

	// Extract infrastructure from load/include paths
	for _, match := range infraLoadPattern.FindAllStringSubmatch(text, -1) {
		infraSet[match[1]] = true
		// Remove from services if it was added
		delete(serviceSet, match[1])
	}

	// Extract local_resource declarations (these are infra/utility tasks)
	for _, match := range localResourcePattern.FindAllStringSubmatch(text, -1) {
		infraSet[match[1]] = true
		delete(serviceSet, match[1])
	}

	// Parse loaded sub-Tiltfiles for additional resources
	for _, match := range loadTiltfilePattern.FindAllStringSubmatch(text, -1) {
		subPath := filepath.Join(tiltfileDir, match[1])
		subContent, err := os.ReadFile(subPath)
		if err != nil {
			continue
		}
		subText := string(subContent)
		isInfra := strings.Contains(match[1], "/infra/")

		for _, subMatch := range k8sResourcePattern.FindAllStringSubmatch(subText, -1) {
			name := subMatch[1]
			if isInfra {
				infraSet[name] = true
				delete(serviceSet, name)
			} else if !infraSet[name] {
				serviceSet[name] = true
			}
		}
	}

	// Build services list with inferred groups
	var services []DiscoveredService
	for name := range serviceSet {
		group := inferGroup(name, text)
		services = append(services, DiscoveredService{Name: name, Group: group})
	}
	sort.Slice(services, func(i, j int) bool {
		return services[i].Name < services[j].Name
	})

	// Build sorted infra list
	var infra []string
	for name := range infraSet {
		infra = append(infra, name)
	}
	sort.Strings(infra)

	return &DiscoveredResources{
		Services: services,
		Infra:    infra,
	}, nil
}

// inferGroup tries to determine the service group from the Tiltfile content.
func inferGroup(serviceName, tiltfileContent string) string {
	// Strategy 1: Helm path pattern
	escapedName := regexp.QuoteMeta(serviceName)
	helmGroupRe := regexp.MustCompile(`k8s/helm/(?:apps|services)/([^/'"]+)/[^'"]*` + escapedName)
	if match := helmGroupRe.FindStringSubmatch(tiltfileContent); len(match) > 1 {
		return match[1]
	}

	// Strategy 2: Labels in Tilt config
	// Look for k8s_resource('serviceName', ... labels=['groupName'])
	resourceBlock := regexp.MustCompile(`k8s_resource\(\s*['"]` + escapedName + `['"][^)]*\)`)
	if block := resourceBlock.FindString(tiltfileContent); block != "" {
		if labelMatch := labelsPattern.FindStringSubmatch(block); len(labelMatch) > 1 {
			// Strip numeric prefixes like "00_auth" -> "auth"
			group := labelMatch[1]
			if idx := strings.Index(group, "_"); idx > 0 {
				prefix := group[:idx]
				if isNumeric(prefix) {
					group = group[idx+1:]
				}
			}
			return group
		}
	}

	return "default"
}

func isNumeric(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(s) > 0
}
