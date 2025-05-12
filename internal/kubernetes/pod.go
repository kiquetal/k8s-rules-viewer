package kubernetes

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/rivo/tview"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// GetPodInfo fetches pod details from the Kubernetes cluster
func GetPodInfo(clientset *kubernetes.Clientset, namespace, podName string) string {
	pod, err := clientset.CoreV1().Pods(namespace).Get(context.TODO(), podName, metav1.GetOptions{})
	if err != nil {
		return fmt.Sprintf("Error retrieving pod: %v", err)
	}

	info := fmt.Sprintf("Name: %s\nNamespace: %s\nStatus: %s\nNode: %s\nIP: %s\n",
		pod.Name,
		pod.Namespace,
		pod.Status.Phase,
		pod.Spec.NodeName,
		pod.Status.PodIP)

	return info
}

// GetPodInfoByLabel fetches pod details using a label selector
func GetPodInfoByLabel(clientset *kubernetes.Clientset, namespace, labelSelector string) []string {
	pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: labelSelector,
	})

	if err != nil {
		return []string{fmt.Sprintf("Error retrieving pods: %v", err)}
	}

	if len(pods.Items) == 0 {
		return []string{"No pods found with the specified label"}
	}

	results := make([]string, len(pods.Items))

	for i, pod := range pods.Items {
		results[i] = fmt.Sprintf("Name: %s\nNamespace: %s\nStatus: %s\nNode: %s\nIP: %s\n",
			pod.Name,
			pod.Namespace,
			pod.Status.Phase,
			pod.Spec.NodeName,
			pod.Status.PodIP)
	}

	return results
}

// GetPodNamesByLabel returns a slice of pod names that match the given label selector
func GetPodNamesByLabel(clientset *kubernetes.Clientset, namespace, labelSelector string) []string {
	pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: labelSelector,
	})

	if err != nil {
		return []string{}
	}

	var podNames []string
	for _, pod := range pods.Items {
		podNames = append(podNames, pod.Name)
	}

	return podNames
}

// GetPodContainers retrieves the list of container names in a pod
func GetPodContainers(clientset *kubernetes.Clientset, namespace, podName string) ([]string, error) {
	pod, err := clientset.CoreV1().Pods(namespace).Get(context.TODO(), podName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("error retrieving pod: %v", err)
	}

	// Sort containers to prioritize app containers over istio/sidecars
	var appContainers []string
	var sidecarContainers []string
	var initContainers []string

	// Categorize regular containers
	for _, container := range pod.Spec.Containers {
		name := container.Name
		if name == "istio-proxy" || name == "envoy" || name == "linkerd" {
			sidecarContainers = append(sidecarContainers, name)
		} else {
			appContainers = append(appContainers, name)
		}
	}

	// Add init containers
	for _, container := range pod.Spec.InitContainers {
		initContainers = append(initContainers, container.Name)
	}

	// Return containers in preferred order: app containers first, then sidecars, then init containers
	result := append(appContainers, sidecarContainers...)
	result = append(result, initContainers...)

	return result, nil
}

// GetPodLogs retrieves logs from a pod's container
func GetPodLogs(clientset *kubernetes.Clientset, namespace, podName string, tailLines int64, containerName string) (string, error) {
	// If no container specified, get container names and try to find the most appropriate one
	if containerName == "" {
		containers, err := GetPodContainers(clientset, namespace, podName)
		if err != nil {
			return "", err
		}

		if len(containers) == 0 {
			return "", fmt.Errorf("no containers found in pod %s", podName)
		}

		// Use the first container (which should be an app container based on GetPodContainers sorting)
		containerName = containers[0]
	}

	podLogOptions := corev1.PodLogOptions{
		TailLines: &tailLines,
		Container: containerName,
	}

	req := clientset.CoreV1().Pods(namespace).GetLogs(podName, &podLogOptions)
	podLogs, err := req.Stream(context.TODO())
	if err != nil {
		return "", fmt.Errorf("error opening log stream: %v", err)
	}
	defer podLogs.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return "", fmt.Errorf("error copying logs: %v", err)
	}

	return buf.String(), nil
}

// RenderPod renders the pod details in the TUI for pods matching the label selector
func RenderPod(clientset *kubernetes.Clientset, app *tview.Application, namespace string, labelSelector string) {
	podInfoList := GetPodInfoByLabel(clientset, namespace, labelSelector)

	// Create a new flex layout for pod information
	flex := tview.NewFlex().SetDirection(tview.FlexRow)

	// Add header
	flex.AddItem(tview.NewTextView().SetText("Pod Monitoring").SetTextAlign(tview.AlignCenter), 1, 0, false)

	if len(podInfoList) == 1 && (podInfoList[0] == "No pods found with the specified label" || podInfoList[0][:5] == "Error") {
		// No pods found or error occurred
		flex.AddItem(tview.NewTextView().SetText(podInfoList[0]), 1, 0, false)
	} else {
		// Display information for each pod
		for i, podInfo := range podInfoList {
			podTextView := tview.NewTextView()
			podTextView.SetBorder(true)
			podTextView.SetTitle(fmt.Sprintf("Pod %d", i+1))
			podTextView.SetText(podInfo)

			flex.AddItem(podTextView, 0, 1, false)

			if i < len(podInfoList)-1 {
				flex.AddItem(tview.NewTextView().SetText(""), 1, 0, false)
			}
		}
	}

	app.SetRoot(flex, true)
}
