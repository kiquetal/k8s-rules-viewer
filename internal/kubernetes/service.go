package kubernetes

import (
	"context"
	"fmt"

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
		portInfo += fmt.Sprintf("- Port: %d, Target Port: %v, Protocol: %s\n",
			port.Port, port.TargetPort.String(), port.Protocol)
	}

	info := fmt.Sprintf("Name: %s\nNamespace: %s\nClusterIP: %s\nType: %s\nSelector: %v\nPorts:\n%s",
		service.Name,
		service.Namespace,
		service.Spec.ClusterIP,
		service.Spec.Type,
		service.Spec.Selector,
		portInfo)

	return info
}

// RenderService renders the service details in the TUI
func RenderService(clientset *kubernetes.Clientset, app *tview.Application, namespace string) {
	// Implementation would go here
	// This is a placeholder to satisfy the compiler as it's referenced in earlier discussions
}
