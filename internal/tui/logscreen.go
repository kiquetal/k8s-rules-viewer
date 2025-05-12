package tui

import (
	"context"
	"fmt"
	"io"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"os"
)

// DisplayLogsInTUI displays logs in the terminal user interface
func DisplayLogsInTUI(clientset *kubernetes.Clientset, namespace, podName, containerName string) {
	fmt.Println("|---------------------------- Pod Logs ------------------------------|")
	// Fetch and stream logs from Kubernetes
	err := StreamPodLogs(clientset, namespace, podName, containerName)
	if err != nil {
		return
	}
}
func StreamPodLogs(clientset *kubernetes.Clientset, namespace, podName, containerName string) error {
	req := clientset.CoreV1().Pods(namespace).GetLogs(podName, &v1.PodLogOptions{
		Container: containerName,
		Follow:    true,
	})

	readCloser, err := req.Stream(context.TODO())
	if err != nil {
		return err
	}
	defer readCloser.Close()

	_, err = io.Copy(os.Stdout, readCloser)
	return err
}
