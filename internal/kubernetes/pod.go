package kubernetes

import (
	"context"
	"fmt"

	"github.com/rivo/tview"
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
