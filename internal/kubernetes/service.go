package kubernetes

import (
	"context"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// GetServiceInfo fetches service details from the Kubernetes cluster
func GetServiceInfo(clientset *kubernetes.Clientset, namespace, serviceName string) string {
	service, err := clientset.CoreV1().Services(namespace).Get(context.TODO(), serviceName, metav1.GetOptions{})
	if err != nil {
		return fmt.Sprintf("Error retrieving service: %v", err)
	}

	portInfo := ""
	for _, port := range service.Spec.Ports {
		// Check if port follows Istio naming conventions
		portNameValid := isValidIstioPortName(port.Name)
		validation := "✓"
		if !portNameValid {
			validation = "✗"
		}

		portInfo += fmt.Sprintf("- Name: %s [%s], Port: %d, Target Port: %v, Protocol: %s\n",
			port.Name, validation, port.Port, port.TargetPort.String(), port.Protocol)
	}

	// Add scrape_tls label info
	scrapeTLS := "false"
	if val, exists := service.Labels["scrape_tls"]; exists && val == "true" {
		scrapeTLS = "true"
	}

	info := fmt.Sprintf("Name: %s\nNamespace: %s\nClusterIP: %s\nType: %s\nSelector: %v\nscrape_tls: %s\nPorts:\n%s",
		service.Name,
		service.Namespace,
		service.Spec.ClusterIP,
		service.Spec.Type,
		service.Spec.Selector,
		scrapeTLS,
		portInfo)

	return info
}

// isValidIstioPortName checks if a port name follows Istio naming conventions
func isValidIstioPortName(portName string) bool {
	if portName == "" {
		return false
	}

	// According to Istio port naming conventions:
	// Port names should have the format <protocol>[-<suffix>]
	validProtocols := []string{"http", "http2", "https", "tcp", "tls", "grpc", "mongo", "redis"}

	for _, protocol := range validProtocols {
		if strings.HasPrefix(portName, protocol) &&
			(len(portName) == len(protocol) || portName[len(protocol)] == '-') {
			return true
		}
	}

	return false
}
