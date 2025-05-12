package main

import (
	"log"
	"os"
	"path/filepath"

	k "github.com/kiquetal/k8s-rules-viewer/internal/kubernetes"
	"github.com/kiquetal/k8s-rules-viewer/internal/tui"
	"github.com/rivo/tview"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	// Load Kubernetes config from default location if not specified
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Fatalf("Error getting user home dir: %v", err)
		}
		kubeconfig = filepath.Join(homeDir, ".kube", "config")
	}

	// Build the Kubernetes config and clientset
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		log.Fatalf("Error building kubeconfig: %s", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Error creating Kubernetes client: %s", err)
	}

	// Create a new tview application
	app := tview.NewApplication()

	// Render the TUI layout with dynamic data
	renderTUI(app, clientset)

	// Run the application
	if err := app.Run(); err != nil {
		log.Fatalf("Error running the application: %v", err)
	}
}

// renderTUI will render the dashboard with dynamic data fetched from Kubernetes
func renderTUI(app *tview.Application, clientset *kubernetes.Clientset) {
	// Create the main layout (using Flex to organize the UI)
	mainFlex := tview.NewFlex().SetDirection(tview.FlexRow)

	// Add the header (title)
	header := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetText("k8s-viewer-rules - Label: app-py-kannel")
	mainFlex.AddItem(header, 3, 0, false)

	// Fetch dynamic Deployment, Service, and Pod Info
	deploymentInfo := k.GetDeploymentInfo(clientset, "default", "py-kannel")
	serviceInfo := k.GetServiceInfo(clientset, "default", "py-kannel-service")
	podInfo := k.GetPodInfo(clientset, "default", "py-kannel-pod")

	// Create content layout (deployment, service, pod info displayed side by side)
	contentFlex := tview.NewFlex().SetDirection(tview.FlexColumn)

	// Deployment Info Section
	deploymentTextView := tview.NewTextView().
		SetBorder(true).
		SetTitle("Deployment Details").
		SetText(deploymentInfo)
	contentFlex.AddItem(deploymentTextView, 0, 1, false)

	// Service Info Section
	serviceTextView := tview.NewTextView().
		SetBorder(true).
		SetTitle("Service Details").
		SetText(serviceInfo)
	contentFlex.AddItem(serviceTextView, 0, 1, false)

	// Pod Info Section
	podTextView := tview.NewTextView().
		SetBorder(true).
		SetTitle("Pod Monitoring").
		SetText(podInfo)
	contentFlex.AddItem(podTextView, 0, 1, false)

	// Add content section to the main layout
	mainFlex.AddItem(contentFlex, 0, 1, true)

	// Rules Compliance Section (hardcoded example, to be replaced with dynamic logic)
	rulesCompliance := tui.GetRulesCompliance() // Assume you fetch rules dynamically in tui package
	rulesTextView := tview.NewTextView().
		SetBorder(true).
		SetTitle("Rules Compliance").
		SetText(rulesCompliance)
	mainFlex.AddItem(rulesTextView, 0, 1, false)

	// Krakend Config Check Section (hardcoded example, to be replaced with dynamic logic)
	krakendConfigCheck := tui.GetKrakendConfigCheck() // Assume you fetch Krakend info dynamically in tui package
	krakendTextView := tview.NewTextView().
		SetBorder(true).
		SetTitle("Krakend Config Check").
		SetText(krakendConfigCheck)
	mainFlex.AddItem(krakendTextView, 0, 1, false)

	// Pod Logs Section (hardcoded example, to be replaced with dynamic logic)
	podLogs := tui.GetPodLogs() // Assume you fetch logs dynamically in tui package
	podLogsTextView := tview.NewTextView().
		SetBorder(true).
		SetTitle("Pod Logs").
		SetText(podLogs)
	mainFlex.AddItem(podLogsTextView, 0, 1, false)

	// Set the root layout and render the TUI
	app.SetRoot(mainFlex, true)
}
