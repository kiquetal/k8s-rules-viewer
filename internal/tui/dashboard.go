package tui

import (
	k "github.com/kiquetal/k8s-rules-viewer/internal/kubernetes"
	"github.com/rivo/tview"
	"k8s.io/client-go/kubernetes"
)

func RenderDashboard(clientset *kubernetes.Clientset, app *tview.Application, namespace string) {
	// Create a new flex layout for the dashboard
	flex := tview.NewFlex().SetDirection(tview.FlexRow)

	// Add title
	title := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetText("Kubernetes Dashboard")

	flex.AddItem(title, 0, 1, false)

	// Add buttons for different sections
	buttons := []struct {
		label  string
		action func()
	}{
		{"Pod Monitoring", func() {
			// Create a form to input label selector
			form := tview.NewForm()
			form.AddInputField("Label Selector (e.g. app=myapp):", "", 50, nil, nil)
			form.AddButton("Submit", func() {
				labelSelector := form.GetFormItem(0).(*tview.InputField).GetText()
				k.RenderPod(clientset, app, namespace, labelSelector)
			})
			form.AddButton("Cancel", func() {
				RenderDashboard(clientset, app, namespace)
			})

			app.SetRoot(form, true)
		}},
		{"Service Monitoring", func() { k.RenderService(clientset, app, namespace) }},
	}

	for _, button := range buttons {
		btn := tview.NewButton(button.label).SetSelectedFunc(button.action)
		flex.AddItem(btn, 0, 1, false)
	}

	app.SetRoot(flex, true)
}
