package kubernetes

import (
	"context"
	"fmt"
	"github.com/rivo/tview"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// GetPodInfo fetches pod details from the Kubernetes cluster
func GetPodInfo(clientset *kubernetes.Clientset, namespace, podName string) (string, string) {
	pod, err := clientset.CoreV1().Pods(namespace).Get(context.TODO(), podName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Sprintf("Error retrieving pod: %v", err)
	}
	return pod.Name, fmt.Sprintf("Status: %s", pod.Status.Phase)
}

// GetPodInfoByLabel fetches pod details using a label selector
func GetPodInfoByLabel(clientset *kubernetes.Clientset, namespace, labelSelector string) ([]string, []string) {
	pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: labelSelector,
	})

	if err != nil {
		return []string{}, []string{fmt.Sprintf("Error retrieving pods: %v", err)}
	}

	if len(pods.Items) == 0 {
		return []string{}, []string{"No pods found with the specified label"}
	}

	names := make([]string, len(pods.Items))
	statuses := make([]string, len(pods.Items))

	for i, pod := range pods.Items {
		names[i] = pod.Name
		statuses[i] = fmt.Sprintf("Status: %s", pod.Status.Phase)
	}

	return names, statuses
}

// RenderPod renders the pod details in the TUI
func RenderPod(clientset *kubernetes.Clientset, app *tview.Application, namespace string, labelSelector string) {
	names, statuses := GetPodInfoByLabel(clientset, namespace, labelSelector)

	// Create a new flex layout for pod information
	flex := tview.NewFlex().SetDirection(tview.FlexRow)

	// Add header
	flex.AddItem(tview.NewTextView().SetText("Pod Monitoring").SetTextAlign(tview.AlignCenter), 1, 0, false)

	if len(names) == 0 {
		// No pods found or error occurred
		flex.AddItem(tview.NewTextView().SetText(statuses[0]), 1, 0, false)
	} else {
		// Display information for each pod
		for i := range names {
			podFlex := tview.NewFlex().SetDirection(tview.FlexRow)
			podFlex.AddItem(tview.NewTextView().SetText(fmt.Sprintf("Name: %s", names[i])), 1, 0, false)
			podFlex.AddItem(tview.NewTextView().SetText(statuses[i]), 1, 0, false)

			if i < len(names)-1 {
				podFlex.AddItem(tview.NewTextView().SetText("---"), 1, 0, false)
			}

			flex.AddItem(podFlex, 0, 1, false)
		}
	}

	app.SetRoot(flex, true)
}
