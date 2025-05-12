package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/gdamore/tcell/v2"
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

var podSelectionHandler func(option string, optionIndex int) // ← DECLARE FIRST clearly

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

	// Get the actual pod names for log retrieval
	podNames := k.GetPodNamesByLabel(clientset, namespace, labelSelector)

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

	// Pod Logs Section - now handling multiple containers
	podLogsTextView := tview.NewTextView()
	podLogsTextView.SetBorder(true)
	podLogsTextView.SetScrollable(true)
	podLogsTextView.SetDynamicColors(true)

	// Check if we have any pods to display logs from
	if len(podNames) > 0 {
		// Get the first pod's name for initial log display
		currentPodName := podNames[0]

		// Create a flex container for the pod selector, container selector, and logs
		podLogsContainer := tview.NewFlex().SetDirection(tview.FlexRow)

		// Create a form for pod and container selection
		podSelectForm := tview.NewForm()

		// Initialize selected container to match app label (most likely the main container)
		preferredContainer := appLabel

		// Get available containers for the selected pod
		containers, err := k.GetPodContainers(clientset, namespace, currentPodName)
		if err != nil {
			podLogsTextView.SetText(fmt.Sprintf("Error retrieving containers: %v", err))
		}

		// Find the initial container to select (try to match the app label first)
		initialContainerIndex := 0
		currentContainer := ""

		// Try to find a container matching the app label first
		for i, c := range containers {
			if strings.Contains(c, preferredContainer) {
				initialContainerIndex = i
				currentContainer = c
				break
			}
		}

		// If no matching container found, try to find a container that's not a sidecar
		if currentContainer == "" {
			for i, c := range containers {
				if c != "istio-proxy" && !strings.Contains(c, "istio") &&
					!strings.HasPrefix(c, "envoy") && !strings.HasPrefix(c, "linkerd") {
					initialContainerIndex = i
					currentContainer = c
					break
				}
			}
		}

		// If still no container found, use the first one if available
		if currentContainer == "" && len(containers) > 0 {
			initialContainerIndex = 0
			currentContainer = containers[0]
		}

		// Define the pod selection handler function so we can use it twice
		podSelectionHandler = func(option string, optionIndex int) {
			selectedPod := podNames[optionIndex]
			currentPodName = selectedPod

			// Update container list when pod changes
			containers, err := k.GetPodContainers(clientset, namespace, selectedPod)
			if err != nil {
				podLogsTextView.SetText(fmt.Sprintf("Error retrieving containers: %v", err))
				return
			}

			// Clear and rebuild the form
			podSelectForm.Clear(true)

			// Re-add pod dropdown - FIXED: pass the podSelectionHandler as the selected function
			podSelectForm.AddDropDown("Pod:", podNames, optionIndex, podSelectionHandler)

			// Find the best container to show for this pod
			newContainerIndex := 0
			currentContainer = ""

			// Try to match by app label first
			for i, c := range containers {
				if strings.Contains(c, preferredContainer) {
					newContainerIndex = i
					currentContainer = c
					break
				}
			}

			// If no match by label, try to find a non-sidecar
			if currentContainer == "" {
				for i, c := range containers {
					if c != "istio-proxy" && !strings.Contains(c, "istio") &&
						!strings.HasPrefix(c, "envoy") && !strings.HasPrefix(c, "linkerd") {
						newContainerIndex = i
						currentContainer = c
						break
					}
				}
			}

			// If still no container found, use the first one
			if currentContainer == "" && len(containers) > 0 {
				newContainerIndex = 0
				currentContainer = containers[0]
			}

			// Add container dropdown with updated containers
			podSelectForm.AddDropDown("Container:", containers, newContainerIndex, func(option string, containerIndex int) {
				selectedContainer := containers[containerIndex]
				currentContainer = selectedContainer
				podLogsTextView.SetTitle(fmt.Sprintf("Pod Logs (%s:%s)", currentPodName, currentContainer))

				// Fetch logs for selected pod and container
				logs, err := k.GetPodLogs(clientset, namespace, currentPodName, 100, currentContainer)
				if err != nil {
					podLogsTextView.SetText(fmt.Sprintf("Error retrieving logs: %v", err))
				} else {
					podLogsTextView.SetText(logs)
				}
			})

			// Update logs with the newly selected container
			if currentContainer != "" {
				podLogsTextView.SetTitle(fmt.Sprintf("Pod Logs (%s:%s)", currentPodName, currentContainer))

				logs, err := k.GetPodLogs(clientset, namespace, currentPodName, 100, currentContainer)
				if err != nil {
					podLogsTextView.SetText(fmt.Sprintf("Error retrieving logs: %v", err))
				} else {
					podLogsTextView.SetText(logs)
				}
			}
		}

		// Add pod selector dropdown with the handler we defined above
		podSelectForm.AddDropDown("Pod:", podNames, 0, podSelectionHandler)

		// Add initial container dropdown if containers are available
		if len(containers) > 0 {
			podSelectForm.AddDropDown("Container:", containers, initialContainerIndex, func(option string, containerIndex int) {
				selectedContainer := containers[containerIndex]
				currentContainer = selectedContainer
				podLogsTextView.SetTitle(fmt.Sprintf("Pod Logs (%s:%s)", currentPodName, currentContainer))

				// Fetch logs for selected pod and container
				logs, err := k.GetPodLogs(clientset, namespace, currentPodName, 100, currentContainer)
				if err != nil {
					podLogsTextView.SetText(fmt.Sprintf("Error retrieving logs: %v", err))
				} else {
					podLogsTextView.SetText(logs)
				}
			})

			// Initial logs load with selected container
			podLogsTextView.SetTitle(fmt.Sprintf("Pod Logs (%s:%s)", currentPodName, currentContainer))

			logs, err := k.GetPodLogs(clientset, namespace, currentPodName, 100, currentContainer)
			if err != nil {
				podLogsTextView.SetText(fmt.Sprintf("Error retrieving logs: %v", err))
			} else {
				podLogsTextView.SetText(logs)
			}
		} else {
			podLogsTextView.SetTitle(fmt.Sprintf("Pod Logs (%s)", currentPodName))
			podLogsTextView.SetText("No containers found in pod")
		}

		podLogsContainer.AddItem(podSelectForm, 3, 0, false)
		podLogsContainer.AddItem(podLogsTextView, 0, 1, true)

		podLogsContainer.SetBorder(true)
		podLogsContainer.SetTitle(fmt.Sprintf("Pod Logs"))

		mainFlex.AddItem(podLogsContainer, 0, 1, false)
	} else {
		// No pods found
		podLogsTextView.SetTitle("Pod Logs")
		podLogsTextView.SetText("No pods found with the specified label")
		mainFlex.AddItem(podLogsTextView, 0, 1, false)
	}

	// Add help text at the bottom
	helpText := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetText("Use arrow keys to navigate and scroll. Press Tab to switch focus. Press F5 to refresh logs. Press Ctrl+C to exit.")
	mainFlex.AddItem(helpText, 1, 0, false)

	// Set up key bindings for refreshing logs
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyF5 {
			// If pods are available, refresh logs for current selection
			if len(podNames) > 0 {
				// Parse title to get current pod and container
				title := podLogsTextView.GetTitle()
				if !strings.HasPrefix(title, "Pod Logs (") || !strings.HasSuffix(title, ")") {
					return event
				}

				info := strings.TrimPrefix(title, "Pod Logs (")
				info = strings.TrimSuffix(info, ")")

				parts := strings.Split(info, ":")

				var currentPodName, currentContainer string
				if len(parts) > 1 {
					currentPodName = parts[0]
					currentContainer = parts[1]
				} else {
					currentPodName = info
					currentContainer = ""
				}

				logs, err := k.GetPodLogs(clientset, namespace, currentPodName, 100, currentContainer)
				if err != nil {
					podLogsTextView.SetText(fmt.Sprintf("Error retrieving logs: %v", err))
				} else {
					podLogsTextView.SetText(logs)
				}
			}
			return nil // Consume the key event
		}
		return event // Pass other keys through
	})

	// Set the root layout and render the TUI
	app.SetRoot(mainFlex, true)
}
