package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	k "github.com/kiquetal/k8s-rules-viewer/internal/kubernetes"
	"github.com/kiquetal/k8s-rules-viewer/internal/tui"
	"github.com/rivo/tview"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	// Define command-line flags for app label, namespace, and krakend config map name
	appLabel := flag.String("label", "py-kannel", "Application label to filter resources")
	namespace := flag.String("namespace", "default", "Kubernetes namespace to search in")
	krakendConfigMap := flag.String("krakend-map", "krakend-config", "Name of the Krakend ConfigMap to look for")

	// Parse command-line flags
	flag.Parse()

	// Display the parameters being used
	fmt.Printf("Using parameters:\n  Label: %s\n  Namespace: %s\n  Krakend ConfigMap: %s\n",
		*appLabel, *namespace, *krakendConfigMap)

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

	// Render the TUI layout with dynamic data using the provided parameters
	renderTUI(app, clientset, *appLabel, *namespace, *krakendConfigMap)

	// Run the application
	if err := app.Run(); err != nil {
		log.Fatalf("Error running the application: %v", err)
	}
}

// renderTUI will render the dashboard with dynamic data fetched from Kubernetes
func renderTUI(app *tview.Application, clientset *kubernetes.Clientset, appLabel, namespace, krakendMap string) {
	// Create the main layout (using Flex to organize the UI)
	mainFlex := tview.NewFlex().SetDirection(tview.FlexRow)

	// Add the header (title) with dynamic parameters
	header := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetText(fmt.Sprintf("k8s-viewer-rules - Label: %s - Namespace: %s", appLabel, namespace))
	mainFlex.AddItem(header, 3, 0, false)

	// Fetch dynamic Deployment, Service, and Pod Info using the provided parameters
	deploymentInfo := k.GetDeploymentInfo(clientset, namespace, appLabel)
	serviceInfo := k.GetServiceInfo(clientset, namespace, appLabel)

	// Use GetPodInfoByLabel to get pods by label instead of by pod name
	// Create the label selector (format: "key=value")
	labelSelector := fmt.Sprintf("app=%s", appLabel)
	podInfoList := k.GetPodInfoByLabel(clientset, namespace, labelSelector)

	// Format the pod information into a single string for display
	var podInfoBuilder strings.Builder
	podInfoBuilder.WriteString(fmt.Sprintf("Pods with label '%s':\n\n", labelSelector))

	for i, podInfo := range podInfoList {
		podInfoBuilder.WriteString(fmt.Sprintf("--- Pod %d ---\n%s\n", i+1, podInfo))
	}

	podInfo := podInfoBuilder.String()

	// Create content layout (deployment, service, pod info displayed side by side)
	contentFlex := tview.NewFlex().SetDirection(tview.FlexColumn)

	// Deployment Info Section
	deploymentTextView := tview.NewTextView()
	deploymentTextView.SetBorder(true)
	deploymentTextView.SetTitle("Deployment Details")
	deploymentTextView.SetText(deploymentInfo)
	deploymentTextView.SetScrollable(true)
	contentFlex.AddItem(deploymentTextView, 0, 1, false)

	// Service Info Section
	serviceTextView := tview.NewTextView()
	serviceTextView.SetBorder(true)
	serviceTextView.SetTitle("Service Details")
	serviceTextView.SetText(serviceInfo)
	serviceTextView.SetScrollable(true)
	contentFlex.AddItem(serviceTextView, 0, 1, false)

	// Pod Info Section - now using the combined information from all pods with scrolling
	podTextView := tview.NewTextView()
	podTextView.SetBorder(true)
	podTextView.SetTitle(fmt.Sprintf("Pod Monitoring (label: %s)", labelSelector))
	podTextView.SetText(podInfo)
	podTextView.SetScrollable(true) // Enable scrolling
	podTextView.SetDynamicColors(true)
	contentFlex.AddItem(podTextView, 0, 1, true) // Make this section focused for scrolling

	// Add content section to the main layout
	mainFlex.AddItem(contentFlex, 0, 1, true)

	// Rules Compliance Section (hardcoded example, to be replaced with dynamic logic)
	rulesCompliance := tui.GetRulesCompliance()
	rulesTextView := tview.NewTextView()
	rulesTextView.SetBorder(true)
	rulesTextView.SetTitle("Rules Compliance")
	rulesTextView.SetText(rulesCompliance)
	rulesTextView.SetScrollable(true) // Enable scrolling
	mainFlex.AddItem(rulesTextView, 0, 1, false)

	// Krakend Config Check Section - now using the actual function to analyze the ConfigMap
	krakendConfigCheck, err := tui.KrakenDBackendServiceCheck(clientset, namespace, krakendMap, appLabel)
	if err != nil {
		krakendConfigCheck = fmt.Sprintf("Error analyzing Krakend ConfigMap: %v", err)
	}

	krakendTextView := tview.NewTextView()
	krakendTextView.SetBorder(true)
	krakendTextView.SetTitle(fmt.Sprintf("Krakend Config Check (%s)", krakendMap))
	krakendTextView.SetText(krakendConfigCheck)
	krakendTextView.SetScrollable(true)
	mainFlex.AddItem(krakendTextView, 0, 1, false)

	// Pod Logs Section (hardcoded example, to be replaced with dynamic logic)
	podLogs := tui.GetPodLogsScreen()
	podLogsTextView := tview.NewTextView()
	podLogsTextView.SetBorder(true)
	podLogsTextView.SetTitle("Pod Logs")
	podLogsTextView.SetText(podLogs)
	podLogsTextView.SetScrollable(true) // Enable scrolling
	mainFlex.AddItem(podLogsTextView, 0, 1, false)

	// Add help text at the bottom
	helpText := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetText("Use arrow keys to navigate and scroll. Press Tab to switch focus. Press Ctrl+C to exit.")
	mainFlex.AddItem(helpText, 1, 0, false)

	// Set the root layout and render the TUI
	app.SetRoot(mainFlex, true)
}
