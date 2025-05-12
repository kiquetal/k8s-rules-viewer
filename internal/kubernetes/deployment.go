package kubernetes

import (
	"context"
	"fmt"

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

	return info
}
