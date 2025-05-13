package tui

import (
	"os"
	"strings"

	"context"
	"fmt"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// StatusSymbols provides both emoji and text fallbacks for statuses
type StatusSymbols struct {
	Success string
	Failure string
}

// GetStatusSymbols returns appropriate status symbols based on terminal capabilities
func GetStatusSymbols() StatusSymbols {
	// Check if terminal likely supports emoji
	// TERM_PROGRAM environment variable is set by many terminal emulators
	termProgram := os.Getenv("TERM_PROGRAM")
	colorTerm := os.Getenv("COLORTERM")
	term := os.Getenv("TERM")

	// These terminals generally have good Unicode/emoji support
	goodTerms := []string{"iTerm.app", "Apple_Terminal", "vscode", "hyper", "alacritty", "kitty", "terminator"}

	useEmoji := false

	// Check if we're in a terminal with likely emoji support
	if termProgram != "" {
		for _, t := range goodTerms {
			if strings.Contains(strings.ToLower(termProgram), strings.ToLower(t)) {
				useEmoji = true
				break
			}
		}
	}

	// Additional checks for terminal types that might support emoji
	if !useEmoji && (strings.Contains(colorTerm, "truecolor") ||
		strings.Contains(colorTerm, "24bit") ||
		strings.Contains(term, "xterm-256color")) {
		useEmoji = true
	}

	if useEmoji {
		return StatusSymbols{
			Success: "✅",
			Failure: "❌",
		}
	}

	// Fallback to ASCII symbols
	return StatusSymbols{
		Success: "[+]",
		Failure: "[!]",
	}
}

// RuleResult represents the result of a rule validation
type RuleResult struct {
	Name        string
	Description string
	Passed      bool
}

// ValidatePodServiceAccount checks if pod has a serviceAccountName (required for mTLS)
func ValidatePodServiceAccount(pod *corev1.Pod) bool {
	return pod != nil && pod.Spec.ServiceAccountName != ""
}

// ValidateDeploymentLabels checks if deployment has required labels
func ValidateDeploymentLabels(deployment *appsv1.Deployment) bool {
	if deployment == nil || len(deployment.Labels) == 0 {
		return false
	}

	// Check for required labels (app and version are commonly required)
	requiredLabels := []string{"app", "version"}
	for _, label := range requiredLabels {
		if _, exists := deployment.Labels[label]; !exists {
			return false
		}
	}

	return true
}

// ValidateServicePortNaming checks if service ports follow Istio naming conventions
func ValidateServicePortNaming(service *corev1.Service) bool {
	if service == nil || len(service.Spec.Ports) == 0 {
		return false
	}

	// According to Istio port naming conventions:
	// Port names should have the format <protocol>[-<suffix>]
	// e.g., http, http-api, tcp-database
	validProtocols := []string{"http", "http2", "https", "tcp", "tls", "grpc", "mongo", "redis"}

	for _, port := range service.Spec.Ports {
		if port.Name == "" {
			// Istio requires named ports
			return false
		}

		// Check if the port name starts with a valid protocol prefix
		validProtocolFound := false
		for _, protocol := range validProtocols {
			if strings.HasPrefix(port.Name, protocol) &&
				(len(port.Name) == len(protocol) || port.Name[len(protocol)] == '-') {
				validProtocolFound = true
				break
			}
		}

		if !validProtocolFound {
			return false
		}
	}

	return true
}

// ValidateServiceHasScrapeTLS checks if the service has the label "scrape_tls = true"
func ValidateServiceHasScrapeTLS(service *corev1.Service) bool {
	if service == nil || service.Labels == nil {
		return false
	}
	val, exists := service.Labels["scrape_tls"]
	return exists && val == "true"
}

// EvaluateRules runs all validation rules against the resources in the namespace
func EvaluateRules(clientset *kubernetes.Clientset, namespace string) []RuleResult {
	results := []RuleResult{}
	ctx := context.TODO()

	// Rule 1: Check if pods have serviceAccountName (for mTLS)
	podList, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	podServiceAccountValid := false
	if err == nil && len(podList.Items) > 0 {
		for _, pod := range podList.Items {
			if ValidatePodServiceAccount(&pod) {
				podServiceAccountValid = true
				break
			}
		}
	}
	results = append(results, RuleResult{
		Name:        "Service Account",
		Description: "Pod has serviceAccountName specified (required for mTLS)",
		Passed:      podServiceAccountValid,
	})

	// Rule 2: Check if deployments have required labels
	deploymentList, err := clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	deploymentLabelsValid := false
	if err == nil && len(deploymentList.Items) > 0 {
		for _, deployment := range deploymentList.Items {
			if ValidateDeploymentLabels(&deployment) {
				deploymentLabelsValid = true
				break
			}
		}
	}
	results = append(results, RuleResult{
		Name:        "Deployment Labels",
		Description: "Deployment has required labels (app, version)",
		Passed:      deploymentLabelsValid,
	})

	// Rule 3: Check only the service selected by app label
	var appLabel string
	if err == nil && len(deploymentList.Items) > 0 {
		// Get app label from the first deployment that has it
		for _, deployment := range deploymentList.Items {
			if label, exists := deployment.Labels["app"]; exists {
				appLabel = label
				break
			}
		}
	}

	servicePortsValid := false
	serviceScrapeTLSValid := false
	if appLabel != "" {
		// Get the service with matching app label
		serviceList, err := clientset.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{
			LabelSelector: "app=" + appLabel,
		})
		if err == nil && len(serviceList.Items) > 0 {
			service := &serviceList.Items[0]
			servicePortsValid = ValidateServicePortNaming(service)
			serviceScrapeTLSValid = ValidateServiceHasScrapeTLS(service)
		}
	}

	results = append(results, RuleResult{
		Name:        "Service Port Naming",
		Description: fmt.Sprintf("Service (app=%s) ports follow Istio naming conventions", appLabel),
		Passed:      servicePortsValid,
	})

	results = append(results, RuleResult{
		Name:        "Service scrape_tls Label",
		Description: fmt.Sprintf("Service (app=%s) has label scrape_tls = true", appLabel),
		Passed:      serviceScrapeTLSValid,
	})

	return results
}
