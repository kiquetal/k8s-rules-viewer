package tui

import (
	"context"
	"fmt"
	"github.com/rivo/tview"
	"io"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"strings"
	_ "sync"
	"time"
)

// DisplayLogsInTUI displays logs in the terminal user interface
func DisplayLogsInTUI(clientset *kubernetes.Clientset, namespace, podName, containerName string, app *tview.Application) {
	// Create a new textview for logs
	logView := tview.NewTextView().
		SetDynamicColors(true).
		SetChangedFunc(func() {
			app.Draw()
		})

	logView.SetBorder(true)
	logView.SetTitle(fmt.Sprintf(" Logs: %s/%s ", podName, containerName))

	// Create a flex layout
	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(logView, 0, 1, true).
		AddItem(tview.NewTextView().
			SetTextAlign(tview.AlignCenter).
			SetText("Press Esc to return"), 1, 0, false)

	// Set this as the root of the application
	app.SetRoot(flex, true)

	// Start streaming logs in a goroutine
	go StreamPodLogsToView(clientset, namespace, podName, containerName, logView)
}

// StreamPodLogsToView streams pod logs to a TextView component
func StreamPodLogsToView(clientset *kubernetes.Clientset, namespace, podName, containerName string, textView *tview.TextView) {
	req := clientset.CoreV1().Pods(namespace).GetLogs(podName, &v1.PodLogOptions{
		Container:  containerName,
		Follow:     true,
		Timestamps: true,
	})

	readCloser, err := req.Stream(context.TODO())
	if err != nil {
		textView.SetText(fmt.Sprintf("Error getting logs: %v", err))
		return
	}
	defer readCloser.Close()

	// Buffer for reading
	buf := make([]byte, 4096)

	for {
		n, err := readCloser.Read(buf)
		if err != nil {
			if err != io.EOF {
				textView.Write([]byte(fmt.Sprintf("\nError reading logs: %v", err)))
			}
			break
		}

		if n > 0 {
			// Format the log entries with colors
			logText := formatLogEntry(string(buf[:n]))

			// Append to the TextView
			fmt.Fprint(textView, logText)
		}
	}
}

// formatLogEntry adds colors and formatting to log entries
func formatLogEntry(entry string) string {
	// Split multi-line entries
	lines := strings.Split(strings.TrimSpace(entry), "\n")
	formattedLines := make([]string, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Extract timestamp if present (assumes standard K8s log format)
		parts := strings.SplitN(line, " ", 2)
		if len(parts) == 2 {
			// Try to parse the timestamp
			timestamp := parts[0]
			content := parts[1]

			// Color code based on log content
			if strings.Contains(strings.ToLower(content), "error") ||
				strings.Contains(strings.ToLower(content), "exception") ||
				strings.Contains(strings.ToLower(content), "fail") {
				formattedLines = append(formattedLines, fmt.Sprintf("[gray]%s[white] [red]%s[white]", timestamp, content))
			} else if strings.Contains(strings.ToLower(content), "warn") {
				formattedLines = append(formattedLines, fmt.Sprintf("[gray]%s[white] [yellow]%s[white]", timestamp, content))
			} else {
				formattedLines = append(formattedLines, fmt.Sprintf("[gray]%s[white] %s", timestamp, content))
			}
		} else {
			formattedLines = append(formattedLines, line)
		}
	}

	return strings.Join(formattedLines, "\n") + "\n"
}

// GetPodLogs returns the most recent logs from a pod as a formatted string
// This function is used to display logs directly in the main UI
func GetPodLogsScreen() string {
	// If you want to make this dynamic in the future, you could modify this
	// function to accept clientset, namespace, pod and container parameters

	// For now, return some sample logs to demonstrate formatting
	sampleLogs := []string{
		time.Now().Add(-35*time.Second).Format(time.RFC3339) + " INFO Starting application",
		time.Now().Add(-30*time.Second).Format(time.RFC3339) + " INFO Connecting to database",
		time.Now().Add(-25*time.Second).Format(time.RFC3339) + " WARN Slow database connection",
		time.Now().Add(-20*time.Second).Format(time.RFC3339) + " INFO Connection established",
		time.Now().Add(-15*time.Second).Format(time.RFC3339) + " ERROR Failed to process request: timeout",
		time.Now().Add(-10*time.Second).Format(time.RFC3339) + " INFO Processing new request",
		time.Now().Add(-5*time.Second).Format(time.RFC3339) + " INFO Request completed successfully",
		time.Now().Format(time.RFC3339) + " INFO System healthy",
	}

	var formattedLogs string
	for _, log := range sampleLogs {
		formattedLogs += formatLogEntry(log)
	}

	return formattedLogs
}

// FetchPodLogs fetches logs from a specific pod and container
// Can be used to make GetPodLogs dynamic in the future
func FetchPodLogs(clientset *kubernetes.Clientset, namespace, podName, containerName string, tailLines int64) (string, error) {
	if clientset == nil {
		return "Kubernetes client not initialized", nil
	}

	podLogOptions := v1.PodLogOptions{
		Container: containerName,
		TailLines: &tailLines,
	}

	req := clientset.CoreV1().Pods(namespace).GetLogs(podName, &podLogOptions)
	podLogs, err := req.Stream(context.TODO())
	if err != nil {
		return "", fmt.Errorf("error opening log stream: %v", err)
	}
	defer podLogs.Close()

	buf := new(strings.Builder)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return "", fmt.Errorf("error copying logs: %v", err)
	}

	return formatLogEntry(buf.String()), nil
}
