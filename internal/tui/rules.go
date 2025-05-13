package tui

import (
	"log"
	"os"
	"strings"

	"context"
	"fmt"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Initialize logger at package level
var debugLog *log.Logger

func init() {
	// Create or append to debug.log file
	logFile, err := os.OpenFile("k8s-rules-viewer-debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return // Silently fail if we can't create log file
	}
	debugLog = log.New(logFile, "", log.LstdFlags)
}

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
func ValidatePodServiceAccount(pod *corev1.Pod, appLabel string) bool {
	if pod == nil || pod.Spec.ServiceAccountName == "" {
		return false
	}
	// Check if serviceAccountName matches the app label value
	if labelValue, exists := pod.Labels["app"]; exists {
		return pod.Spec.ServiceAccountName == labelValue
	}
	return false
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
	if debugLog != nil {
		debugLog.Printf("Validating service ports for service: %s", service.Name)
		debugLog.Printf("Service ports: %+v", service.Spec.Ports)
	}
	if service == nil || len(service.Spec.Ports) == 0 {
		return false
	}

	validProtocols := []string{"http", "http2", "https", "tcp", "tls", "grpc", "mongo", "redis"}

	for _, port := range service.Spec.Ports {
		if port.Name == "" {
			return false
		}

		// Split the port name by "-" and check if the first part is a valid protocol
		portNameParts := strings.Split(strings.ToLower(port.Name), "-")
		if len(portNameParts) == 0 {
			return false
		}

		validProtocolFound := false
		for _, protocol := range validProtocols {
			if portNameParts[0] == protocol {
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
func EvaluateRules(clientset *kubernetes.Clientset, namespace string, appLabel string) []RuleResult {
	if debugLog != nil {
		debugLog.Printf("Starting evaluation with appLabel: %q in namespace: %q", appLabel, namespace)
	}

	results := []RuleResult{}
	ctx := context.TODO()

	// Rule 1: Check if pods have serviceAccountName (for mTLS)
	podList, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: appLabel,
	})
	if debugLog != nil {
		debugLog.Printf("Pod list query result - Error: %v, Count: %d", err, len(podList.Items))
	}
	podServiceAccountValid := false
	if err == nil && len(podList.Items) > 0 {
		for _, pod := range podList.Items {
			if ValidatePodServiceAccount(&pod, appLabel) {
				podServiceAccountValid = true
				break
			}
		}
	}
	results = append(results, RuleResult{
		Name:        "Service Account",
		Description: "Pod serviceAccountName matches app label value",
		Passed:      podServiceAccountValid,
	})

	// Rule 2: Check if deployments have required labels
	deploymentList, err := clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: appLabel,
	})
	if debugLog != nil {
		debugLog.Printf("Deployment list query result - Error: %v, Count: %d", err, len(deploymentList.Items))
	}
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

	servicePortsValid := false
	serviceScrapeTLSValid := false
	if appLabel != "" {
		// Clean the label and get the actual value
		cleanLabel := strings.Trim(strings.TrimPrefix(appLabel, "app="), "\"")

		// Try both app label and argocd instance label
		labelSelectors := []string{
			fmt.Sprintf("app=%s", cleanLabel),
			fmt.Sprintf("argocd.argoproj.io/instance=%s", cleanLabel),
		}

		var service *corev1.Service
		for _, selector := range labelSelectors {
			if debugLog != nil {
				debugLog.Printf("Trying service label selector: %s", selector)
			}

			serviceList, err := clientset.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{
				LabelSelector: selector,
			})
			if debugLog != nil {
				debugLog.Printf("Service list query result for %s - Error: %v, Count: %d",
					selector, err, len(serviceList.Items))
			}

			if err == nil && len(serviceList.Items) > 0 {
				service = &serviceList.Items[0]
				if debugLog != nil {
					debugLog.Printf("Found service: %s with labels: %v", service.Name, service.Labels)
				}
				break
			}
		}

		if service != nil {
			servicePortsValid = ValidateServicePortNaming(service)
			serviceScrapeTLSValid = ValidateServiceHasScrapeTLS(service)
		}
	}

	results = append(results, RuleResult{
		Name:        "Service Port Naming",
		Description: fmt.Sprintf("Service (%s) ports follow Istio naming conventions", appLabel),
		Passed:      servicePortsValid,
	})

	results = append(results, RuleResult{
		Name:        "Service scrape_tls Label",
		Description: fmt.Sprintf("Service (%s) has label scrape_tls = true", appLabel),
		Passed:      serviceScrapeTLSValid,
	})

	return results
}

// GetRulesCompliance evaluates all rules and returns a formatted compliance report string
func GetRulesCompliance(clientset *kubernetes.Clientset, namespace string, appLabel string) string {
	// Evaluate all rules
	results := EvaluateRules(clientset, namespace, appLabel)
	fmt.Printf("app-lable", appLabel)
	// Get appropriate status symbols based on terminal capabilities
	symbols := GetStatusSymbols()

	// Format the results
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Compliance check for namespace: %s\n\n", namespace))

	for _, result := range results {
		symbol := symbols.Failure
		if result.Passed {
			symbol = symbols.Success
		}

		sb.WriteString(fmt.Sprintf("%s %s: %s\n",
			symbol,
			result.Name,
			result.Description))
	}

	return sb.String()
}
