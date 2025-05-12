package tui

import (
	"fmt"
	"k8s.io/client-go/kubernetes"
)

// DisplayLogsInTUI displays logs in the terminal user interface
func DisplayLogsInTUI(clientset *kubernetes.Clientset, namespace, podName, containerName string) {
	fmt.Println("|---------------------------- Pod Logs ------------------------------|")
	// Fetch and stream logs from Kubernetes
	kubernetes.StreamPodLogs(clientset, namespace, podName, containerName)
}
