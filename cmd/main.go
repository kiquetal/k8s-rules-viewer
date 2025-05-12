package main

import (
	"github.com/kiquetal/k8s-rules-viewer/internal/tui"
	"github.com/rivo/tview"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"log"
	"os"
	"path/filepath"
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

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		log.Fatalf("Error building kubeconfig: %s", err)
	}

	// Create Kubernetes clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Error creating Kubernetes client: %s", err)
	}

	// Create a new tview application
	app := tview.NewApplication()

	// Render the TUI sections
	renderTUI(app, clientset)

	// Run the application
	if err := app.Run(); err != nil {
		log.Fatalf("Error running the application: %v", err)
	}
}

// renderTUI will render the dashboard and allow dynamic section changes
func renderTUI(app *tview.Application, clientset *kubernetes.Clientset) {
	// Create a title text view
	title := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetText("Welcome to k8s-rules-viewer!")

	// Create menu buttons with vertical orientation
	menu := tview.NewFlex().SetDirection(tview.FlexRow)

	// Dashboard button
	dashboardButton := tview.NewButton("Main Dashboard").SetSelectedFunc(func() {
		tui.RenderDashboard(clientset, app, "default")
	})
	menu.AddItem(dashboardButton, 3, 0, true)

	// Service info button
	serviceButton := tview.NewButton("Service Info").SetSelectedFunc(func() {
		RenderService(clientset, app, "default", "py-kannel-service")
	})
	menu.AddItem(serviceButton, 3, 0, false)

	// Pod info button
	podButton := tview.NewButton("Pod Info").SetSelectedFunc(func() {
		RenderPod(clientset, app, "default", "py-kannel-pod")
	})
	menu.AddItem(podButton, 3, 0, false)

	// Main layout
	layout := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(title, 3, 0, false).
		AddItem(menu, 0, 1, true)

	app.SetRoot(layout, true).SetFocus(menu)
}
