package kubernetes

import (
	"context"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// GetDeploymentInfo fetches deployment details from the Kubernetes cluster
func GetDeploymentInfo(clientset *kubernetes.Clientset, namespace, deploymentName string) string {
	deployment, err := clientset.AppsV1().Deployments(namespace).Get(context.TODO(), deploymentName, metav1.GetOptions{})
	if err != nil {
		return fmt.Sprintf("Error retrieving deployment: %v", err)
	}

	info := fmt.Sprintf("Name: %s\nNamespace: %s\nReplicas: %d/%d\nCreation Time: %s\nSelector: %v\n",
		deployment.Name,
		deployment.Namespace,
		deployment.Status.ReadyReplicas,
		deployment.Status.Replicas,
		deployment.CreationTimestamp.String(),
		deployment.Spec.Selector.MatchLabels)

	// Add labels information with validation
	if len(deployment.Labels) > 0 {
		labelStrings := []string{"Labels:"}
		requiredLabels := []string{"app", "version"}

		for k, v := range deployment.Labels {
			validation := " "
			// Mark required labels
			for _, reqLabel := range requiredLabels {
				if k == reqLabel {
					validation = "✓"
					break
				}
			}
			labelStrings = append(labelStrings, fmt.Sprintf("  %s: %s [%s]", k, v, validation))
		}

		// Check for missing required labels
		for _, reqLabel := range requiredLabels {
			if _, exists := deployment.Labels[reqLabel]; !exists {
				labelStrings = append(labelStrings, fmt.Sprintf("  %s: MISSING [✗]", reqLabel))
			}
		}

		info += strings.Join(labelStrings, "\n") + "\n"
	} else {
		info += "Labels: None (Missing required labels: app, version) [✗]\n"
	}

	return info
}
