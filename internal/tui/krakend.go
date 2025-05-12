package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// GetKrakendConfigCheck returns information about KrakenD configuration status
func GetKrakendConfigCheck() string {
	// This would be replaced with actual KrakenD configuration checking logic
	return `✅ Config Syntax: Valid
✅ Endpoints: All configured correctly
❌ Rate Limiting: Not configured
✅ JWT Validation: Enabled
✅ Backend Services: All reachable`
}

// KrakenDBackendServiceCheck checks if a service is referenced in KrakenD backend configuration
func KrakenDBackendServiceCheck(clientset *kubernetes.Clientset, namespace, configMapName, serviceName string) (string, error) {
	if clientset == nil {
		return "", fmt.Errorf("kubernetes client not initialized")
	}

	// Get the ConfigMap
	configMap, err := clientset.CoreV1().ConfigMaps(namespace).Get(context.TODO(), configMapName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get ConfigMap %s: %v", configMapName, err)
	}

	// Check if the ConfigMap has the KrakenD configuration data
	krakendConfig, exists := configMap.Data["krakend.json"]
	if !exists {
		// Try common alternative filenames
		for key := range configMap.Data {
			if strings.HasSuffix(key, ".json") {
				krakendConfig = configMap.Data[key]
				break
			}
		}
		if krakendConfig == "" {
			return "", fmt.Errorf("no JSON configuration found in ConfigMap %s", configMapName)
		}
	}

	// Parse the JSON configuration
	var config map[string]interface{}
	if err := json.Unmarshal([]byte(krakendConfig), &config); err != nil {
		return "", fmt.Errorf("failed to parse KrakenD configuration: %v", err)
	}

	// Check for the service in backend configurations
	references := findServiceReferences(config, serviceName)
	if len(references) == 0 {
		return fmt.Sprintf("❌ Service '%s' not found in KrakenD backend configuration", serviceName), nil
	}

	// Build result string with references found
	result := fmt.Sprintf("✅ Service '%s' found in %d backend configurations:\n", serviceName, len(references))
	for i, ref := range references {
		result += fmt.Sprintf("  %d. %s\n", i+1, ref)
	}

	return result, nil
}

// findServiceReferences searches the KrakenD config for service references
// by iterating through endpoints and backends (non-recursive approach)
func findServiceReferences(config interface{}, serviceName string) []string {
	var references []string

	// Check if config is a map and has endpoints
	configMap, ok := config.(map[string]interface{})
	if !ok {
		return references
	}

	// Get the endpoints array
	endpoints, ok := configMap["endpoints"].([]interface{})
	if !ok {
		return references
	}

	// Iterate through each endpoint
	for _, endpoint := range endpoints {
		endpointMap, ok := endpoint.(map[string]interface{})
		if !ok {
			continue
		}

		// Get endpoint path for reference
		endpointPath, _ := endpointMap["endpoint"].(string)
		if endpointPath == "" {
			endpointPath = "unknown"
		}

		// Get the backends array
		backends, ok := endpointMap["backend"].([]interface{})
		if !ok {
			continue
		}

		// Iterate through each backend
		for _, backend := range backends {
			backendMap, ok := backend.(map[string]interface{})
			if !ok {
				continue
			}

			// Check url_pattern for service name
			if url, ok := backendMap["url_pattern"].(string); ok && strings.Contains(url, serviceName) {
				references = append(references,
					fmt.Sprintf("Endpoint: %s → Backend: %s", endpointPath, url))
			}

			// Check host field which could be either a string or an array of strings
			found := false
			switch host := backendMap["host"].(type) {
			case string:
				if strings.Contains(host, serviceName) {
					references = append(references,
						fmt.Sprintf("Endpoint: %s → Host: %s", endpointPath, host))
					found = true
				}
			case []interface{}:
				// Handle the case where host is an array of strings
				for _, h := range host {
					if hostStr, ok := h.(string); ok && strings.Contains(hostStr, serviceName) {
						references = append(references,
							fmt.Sprintf("Endpoint: %s → Host: %s", endpointPath, hostStr))
						found = true
						break // Found in this host array, no need to check further
					}
				}
			}

			// If found in the host, continue to the next backend
			if found {
				continue
			}
		}
	}

	return references
}
