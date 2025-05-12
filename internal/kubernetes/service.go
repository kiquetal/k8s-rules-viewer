package kubernetes

import (
	"context"
	"fmt"
	"github.com/rivo/tview"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// GetServiceInfo fetches service information from the Kubernetes cluster
func GetServiceInfo(clientset *kubernetes.Clientset, namespace, serviceName string) (string, string) {
	service, err := clientset.CoreV1().Services(namespace).Get(context.TODO(), serviceName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Sprintf("Error retrieving service: %v", err)
	}
	return service.Name, fmt.Sprintf("Endpoints: %v", service.Spec.Ports)
}

// RenderService renders the service details in the TUI
func RenderService(clientset *kubernetes.Clientset, app *tview.Application, namespace, serviceName string) {
	name, details := GetServiceInfo(clientset, namespace, serviceName)

	// Create a new flex layout
	flex := tview.NewFlex().
		AddItem(tview.NewTextView().SetText("Service Details").SetTextAlign(tview.AlignCenter), 1, 0, false).
		AddItem(tview.NewTextView().SetText(fmt.Sprintf("Name: %s", name)), 1, 0, false).
		AddItem(tview.NewTextView().SetText(details), 1, 0, false)

	app.SetRoot(flex, true)
}
