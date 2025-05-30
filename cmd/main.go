package main

import (
	"flag"
	"fmt"
	"github.com/gdamore/tcell/v2"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

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

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Handle signals in a separate goroutine
	go func() {
		<-sigChan
		app.Stop()
		fmt.Println("\nShutting down gracefully...")
		os.Exit(0)
	}()

	// Use a simple loading screen until we fetch data
	loadingText := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetText("Loading data from Kubernetes cluster...\nThis may take a few seconds.")
	loadingText.SetBorder(true).SetTitle("Loading")
	app.SetRoot(loadingText, true)

	// Pre-fetch the Kubernetes data in a goroutine to avoid blocking the UI
	go func() {
		// Fix the label selector format - it should match what's actually used in Kubernetes
		labelSelector := fmt.Sprintf("app=%s", *appLabel)
		altLabelSelector := fmt.Sprintf("app.kubernetes.io/name=%s", *appLabel)

		// Try first with our primary selector
		podNames := k.GetPodNamesByLabel(clientset, *namespace, labelSelector)
		podInfoList := k.GetPodInfoByLabel(clientset, *namespace, labelSelector)

		// If no pods found, try with the alternative selector
		if len(podNames) == 0 {
			podNames = k.GetPodNamesByLabel(clientset, *namespace, altLabelSelector)
			podInfoList = k.GetPodInfoByLabel(clientset, *namespace, altLabelSelector)
			if len(podNames) > 0 {
				labelSelector = altLabelSelector // Update if we found pods with this selector
			}
		}

		// If still no pods found, try just matching by the app name without explicit label key
		if len(podNames) == 0 {
			// Try a more permissive selector
			podNames = k.GetPodNamesByLabel(clientset, *namespace, *appLabel)
			podInfoList = k.GetPodInfoByLabel(clientset, *namespace, *appLabel)
			if len(podNames) > 0 {
				labelSelector = *appLabel // Update if we found pods with this selector
			}
		}

		// Fetch dynamic Deployment, Service info
		deploymentInfo := k.GetDeploymentInfo(clientset, *namespace, *appLabel)
		serviceInfo := k.GetServiceInfo(clientset, *namespace, *appLabel)

		// Format the pod information into a single string for display
		var podInfoBuilder strings.Builder
		podInfoBuilder.WriteString(fmt.Sprintf("Pods with label '%s':\n\n", labelSelector))

		for i, podInfo := range podInfoList {
			podInfoBuilder.WriteString(fmt.Sprintf("--- Pod %d ---\n%s\n", i+1, podInfo))
		}

		podInfo := podInfoBuilder.String()

		// Get rules compliance information
		rulesCompliance := tui.GetRulesCompliance(clientset, *namespace, labelSelector)

		// Get Krakend config check information
		krakendConfigCheck, err := tui.KrakenDBackendServiceCheck(clientset, *namespace, *krakendConfigMap, *appLabel)
		if err != nil {
			krakendConfigCheck = fmt.Sprintf("Error analyzing Krakend ConfigMap: %v", err)
		}

		// Update the UI with the fetched data
		app.QueueUpdateDraw(func() {
			renderTUI(app, *appLabel, *namespace, *krakendConfigMap, labelSelector,
				deploymentInfo, serviceInfo, podInfo, rulesCompliance, krakendConfigCheck)
		})
	}()

	// Run the application and handle any errors
	if err := app.Run(); err != nil {
		log.Fatalf("Error running the application: %v", err)
	}

	fmt.Println("Application terminated normally")
}

// renderTUI will render the dashboard with pre-fetched data
func renderTUI(app *tview.Application, appLabel, namespace, krakendMap,
	labelSelector, deploymentInfo, serviceInfo, podInfo, rulesCompliance, krakendConfigCheck string) {

	// Create the main layout (using Flex to organize the UI)
	mainFlex := tview.NewFlex().SetDirection(tview.FlexRow)

	// Add the header (title) with dynamic parameters
	header := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetText(fmt.Sprintf("k8s-viewer-rules - Label: %s - Namespace: %s", appLabel, namespace))
	mainFlex.AddItem(header, 3, 0, false)

	// Create content layout (deployment, service, pod info displayed side by side)
	contentFlex := tview.NewFlex().SetDirection(tview.FlexColumn)

	// Deployment Info Section
	deploymentTextView := tview.NewTextView()
	deploymentTextView.SetBorder(true)
	deploymentTextView.SetTitle("Deployment Details")
	deploymentTextView.SetText(deploymentInfo)
	deploymentTextView.SetScrollable(true)
	contentFlex.AddItem(deploymentTextView, 0, 1, true)

	// Service Info Section
	serviceTextView := tview.NewTextView()
	serviceTextView.SetBorder(true)
	serviceTextView.SetTitle("Service Details")
	serviceTextView.SetText(serviceInfo)
	serviceTextView.SetScrollable(true)
	contentFlex.AddItem(serviceTextView, 0, 1, true)

	// Pod Info Section - now using the combined information from all pods with scrolling
	podTextView := tview.NewTextView()
	podTextView.SetBorder(true)
	podTextView.SetTitle(fmt.Sprintf("Pod Monitoring (label: %s)", labelSelector))
	podTextView.SetText(podInfo)
	podTextView.SetScrollable(true) // Enable scrolling
	podTextView.SetDynamicColors(true)
	contentFlex.AddItem(podTextView, 0, 1, true)

	// Add content section to the main layout
	mainFlex.AddItem(contentFlex, 0, 1, true)

	// Rules Compliance Section
	rulesTextView := tview.NewTextView()
	rulesTextView.SetBorder(true)
	rulesTextView.SetTitle("Rules Compliance")
	rulesTextView.SetText(rulesCompliance)
	rulesTextView.SetScrollable(true)
	mainFlex.AddItem(rulesTextView, 0, 1, true)

	// Krakend Config Check Section
	krakendTextView := tview.NewTextView()
	krakendTextView.SetBorder(true)
	krakendTextView.SetTitle(fmt.Sprintf("Krakend Config Check (%s)", krakendMap))
	krakendTextView.SetText(krakendConfigCheck)
	krakendTextView.SetScrollable(true)
	mainFlex.AddItem(krakendTextView, 0, 1, true)

	// Add help text at the bottom
	helpText := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetText("Use Tab to switch focus between panels. Use arrow keys to scroll content. Press Ctrl+C to exit.")
	mainFlex.AddItem(helpText, 1, 0, false)

	// Store all focusable views in order
	focusableViews := []tview.Primitive{
		deploymentTextView,
		serviceTextView,
		podTextView,
		rulesTextView,
		krakendTextView,
	}

	// Set the initial focus to the first view
	app.SetFocus(deploymentTextView)

	// Track current focus index
	currentFocus := 0

	// Set the root layout and render the TUI
	app.SetRoot(mainFlex, true)

	// Set input capture to handle tab navigation between panels
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyTab {
			// Move to next focusable view
			currentFocus = (currentFocus + 1) % len(focusableViews)
			app.SetFocus(focusableViews[currentFocus])
			return nil
		} else if event.Key() == tcell.KeyBacktab {
			// Move to previous focusable view
			currentFocus = (currentFocus - 1 + len(focusableViews)) % len(focusableViews)
			app.SetFocus(focusableViews[currentFocus])
			return nil
		}
		return event
	})
}
