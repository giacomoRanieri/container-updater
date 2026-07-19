package k8s

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"container-updater/backend/internal/db"
	"container-updater/backend/internal/logger"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type K8sClient struct {
	clientset *kubernetes.Clientset
}

func NewK8sClient() (*K8sClient, error) {
	// Attempt in-cluster config first
	cfg, err := rest.InClusterConfig()
	if err != nil {
		logger.Log.Info("Kubernetes in-cluster config not found, attempting out-of-cluster kubeconfig fallback...")
		
		// Fallback to kubeconfig in HOME directory
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		
		kubeconfig := filepath.Join(home, ".kube", "config")
		cfg, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("failed to build kubeconfig from path %s: %w", kubeconfig, err)
		}
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes clientset: %w", err)
	}

	return &K8sClient{clientset: clientset}, nil
}

func (k *K8sClient) ListMonitoredWorkloads(ctx context.Context) ([]*db.Workload, error) {
	var workloads []*db.Workload
	now := time.Now()

	// 1. Scan Deployments
	deploys, err := k.clientset.AppsV1().Deployments("").List(ctx, metav1.ListOptions{})
	if err == nil {
		for _, dep := range deploys.Items {
			if dep.Annotations["container-updater.enable"] != "true" {
				continue
			}
			for _, container := range dep.Spec.Template.Spec.Containers {
				digest := k.getRunningDigest(ctx, dep.Namespace, "Deployment", dep.Name, container.Name)
				workloads = append(workloads, &db.Workload{
					ID:               fmt.Sprintf("k8s-%s-deploy-%s-%s", dep.Namespace, dep.Name, container.Name),
					Name:             fmt.Sprintf("%s/%s", dep.Name, container.Name),
					NamespaceProject: dep.Namespace,
					OrchestratorType: "kubernetes",
					CurrentImage:     container.Image,
					CurrentDigest:    digest,
					UpdateStatus:     "up_to_date",
					LastCheckedAt:    &now,
				})
			}
		}
	} else {
		logger.Log.Error("failed to list deployments", "error", err)
	}

	// 2. Scan StatefulSets
	stsets, err := k.clientset.AppsV1().StatefulSets("").List(ctx, metav1.ListOptions{})
	if err == nil {
		for _, sts := range stsets.Items {
			if sts.Annotations["container-updater.enable"] != "true" {
				continue
			}
			for _, container := range sts.Spec.Template.Spec.Containers {
				digest := k.getRunningDigest(ctx, sts.Namespace, "StatefulSet", sts.Name, container.Name)
				workloads = append(workloads, &db.Workload{
					ID:               fmt.Sprintf("k8s-%s-sts-%s-%s", sts.Namespace, sts.Name, container.Name),
					Name:             fmt.Sprintf("%s/%s", sts.Name, container.Name),
					NamespaceProject: sts.Namespace,
					OrchestratorType: "kubernetes",
					CurrentImage:     container.Image,
					CurrentDigest:    digest,
					UpdateStatus:     "up_to_date",
					LastCheckedAt:    &now,
				})
			}
		}
	} else {
		logger.Log.Error("failed to list statefulsets", "error", err)
	}

	// 3. Scan DaemonSets
	dmsets, err := k.clientset.AppsV1().DaemonSets("").List(ctx, metav1.ListOptions{})
	if err == nil {
		for _, ds := range dmsets.Items {
			if ds.Annotations["container-updater.enable"] != "true" {
				continue
			}
			for _, container := range ds.Spec.Template.Spec.Containers {
				digest := k.getRunningDigest(ctx, ds.Namespace, "DaemonSet", ds.Name, container.Name)
				workloads = append(workloads, &db.Workload{
					ID:               fmt.Sprintf("k8s-%s-ds-%s-%s", ds.Namespace, ds.Name, container.Name),
					Name:             fmt.Sprintf("%s/%s", ds.Name, container.Name),
					NamespaceProject: ds.Namespace,
					OrchestratorType: "kubernetes",
					CurrentImage:     container.Image,
					CurrentDigest:    digest,
					UpdateStatus:     "up_to_date",
					LastCheckedAt:    &now,
				})
			}
		}
	} else {
		logger.Log.Error("failed to list daemonsets", "error", err)
	}

	return workloads, nil
}

// getRunningDigest fetches the pod statuses for the workload and extracts the actual ImageID registry digest.
func (k *K8sClient) getRunningDigest(ctx context.Context, namespace, workloadType, workloadName, containerName string) string {
	// Query pods matching workload selector/labels or label prefixes
	// To be safe and simple, list pods in the namespace and match by pod owner reference or name prefix
	pods, err := k.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return ""
	}

	for _, pod := range pods.Items {
		// Verify if this pod belongs to our workload
		isOwner := false
		for _, ref := range pod.OwnerReferences {
			if ref.Name == workloadName && ref.Kind == workloadType {
				isOwner = true
				break
			}
			// For Deployments, pod owner is ReplicaSet, whose owner is Deployment.
			if workloadType == "Deployment" && ref.Kind == "ReplicaSet" {
				if strings.HasPrefix(ref.Name, workloadName+"-") {
					isOwner = true
					break
				}
			}
		}

		if !isOwner {
			continue
		}

		// Find the container status
		for _, status := range pod.Status.ContainerStatuses {
			if status.Name == containerName {
				// ImageID is typically: "docker.io/library/nginx@sha256:456c..."
				if strings.Contains(status.ImageID, "sha256:") {
					parts := strings.Split(status.ImageID, "@")
					if len(parts) == 2 {
						return parts[1]
					}
				}
				// Fallback to the shorter image ID
				return status.ImageID
			}
		}
	}

	return ""
}
