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

// findServiceReferences recursively searches the KrakenD config for service references
func findServiceReferences(config interface{}, serviceName string) []string {
	var references []string

	switch v := config.(type) {
	case map[string]interface{}:
		// Check if this is an endpoint with backends
		if endpoints, ok := v["endpoints"].([]interface{}); ok {
			for _, endpoint := range endpoints {
				if endpointMap, ok := endpoint.(map[string]interface{}); ok {
					if backends, ok := endpointMap["backend"].([]interface{}); ok {
						for _, backend := range backends {
							if backendMap, ok := backend.(map[string]interface{}); ok {
								if url, ok := backendMap["url_pattern"].(string); ok {
									if strings.Contains(url, serviceName) {
										endpoint := "unknown"
										if ep, ok := endpointMap["endpoint"].(string); ok {
											endpoint = ep
										}
										references = append(references,
											fmt.Sprintf("Endpoint: %s → Backend: %s", endpoint, url))
									}
								}
								// Also check host field which might contain service references
								if host, ok := backendMap["host"].(string); ok {
									if strings.Contains(host, serviceName) {
										endpoint := "unknown"
										if ep, ok := endpointMap["endpoint"].(string); ok {
											endpoint = ep
										}
										references = append(references,
											fmt.Sprintf("Endpoint: %s → Host: %s", endpoint, host))
									}
								}
							}
						}
					}
				}
				// Recursively check this endpoint
				refs := findServiceReferences(endpoint, serviceName)
				references = append(references, refs...)
			}
		}

		// Recursively check all other fields
		for _, val := range v {
			refs := findServiceReferences(val, serviceName)
			references = append(references, refs...)
		}

	case []interface{}:
		// Search through array elements
		for _, item := range v {
			refs := findServiceReferences(item, serviceName)
			references = append(references, refs...)
		}
	}

	return references
}
