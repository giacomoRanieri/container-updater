package k8s

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"container-updater/backend/internal/logger"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ProgressCallback is called during rollout to report status updates
type ProgressCallback func(percent int, message string, podStatus string)

// WatchRollout monitors a Kubernetes workload rollout until completion or timeout.
func WatchRollout(ctx context.Context, namespace, workloadType, workloadName string, timeout time.Duration, callback ProgressCallback) error {
	kClient, err := NewK8sClient()
	if err != nil {
		logger.Log.Warn("Kubernetes client unavailable for rollout monitoring (skipped)", "error", err)
		if callback != nil {
			callback(100, "In-cluster client unavailable, skipping rollout wait.", "Completed")
		}
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	workloadTypeLower := strings.ToLower(workloadType)
	logger.Log.Info("Starting k8s rollout watcher", "namespace", namespace, "type", workloadType, "name", workloadName, "timeout", timeout.String())

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("rollout timeout exceeded (%s) for %s/%s", timeout.String(), workloadType, workloadName)
		case <-ticker.C:
			complete, percent, msg, podStatus, err := checkWorkloadRolloutStatus(ctx, kClient, namespace, workloadTypeLower, workloadName)
			if err != nil {
				return err
			}

			if callback != nil {
				callback(percent, msg, podStatus)
			}

			if complete {
				logger.Log.Info("Rollout completed successfully", "namespace", namespace, "workload", workloadName)
				return nil
			}
		}
	}
}

func checkWorkloadRolloutStatus(ctx context.Context, kClient *K8sClient, namespace, workloadTypeLower, workloadName string) (complete bool, percent int, msg string, podStatus string, err error) {
	// Inspect pods for crashing or image pull issues
	pods, _ := kClient.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if pods != nil {
		for _, pod := range pods.Items {
			if strings.HasPrefix(pod.Name, workloadName) {
				for _, cs := range pod.Status.ContainerStatuses {
					if cs.State.Waiting != nil {
						reason := cs.State.Waiting.Reason
						if reason == "ImagePullBackOff" || reason == "ErrImagePull" || reason == "CrashLoopBackOff" {
							return false, 0, fmt.Sprintf("Pod %s failed: %s - %s", pod.Name, reason, cs.State.Waiting.Message), reason, fmt.Errorf("rollout failed: pod %s entered %s", pod.Name, reason)
						}
						podStatus = reason
					} else if cs.State.Terminated != nil && cs.State.Terminated.ExitCode != 0 {
						reason := cs.State.Terminated.Reason
						return false, 0, fmt.Sprintf("Pod %s terminated with exit code %d (%s)", pod.Name, cs.State.Terminated.ExitCode, reason), "Terminated", fmt.Errorf("rollout failed: pod %s terminated with exit code %d", pod.Name, cs.State.Terminated.ExitCode)
					} else if cs.State.Running != nil {
						podStatus = "Running"
					}
				}
			}
		}
	}

	if podStatus == "" {
		podStatus = "Pending"
	}

	switch workloadTypeLower {
	case "deployment":
		deploy, err := kClient.clientset.AppsV1().Deployments(namespace).Get(ctx, workloadName, metav1.GetOptions{})
		if err != nil {
			return false, 0, "", "", fmt.Errorf("failed to get deployment %s: %w", workloadName, err)
		}

		desiredReplicas := int32(1)
		if deploy.Spec.Replicas != nil {
			desiredReplicas = *deploy.Spec.Replicas
		}

		updatedReplicas := deploy.Status.UpdatedReplicas
		readyReplicas := deploy.Status.ReadyReplicas
		availableReplicas := deploy.Status.AvailableReplicas

		// Check for ProgressDeadlineExceeded
		for _, cond := range deploy.Status.Conditions {
			if cond.Type == "Progressing" && cond.Reason == "ProgressDeadlineExceeded" {
				return false, 0, "Deployment progress deadline exceeded", "Failed", fmt.Errorf("deployment %s progress deadline exceeded", workloadName)
			}
		}

		if updatedReplicas == desiredReplicas && readyReplicas == desiredReplicas && availableReplicas == desiredReplicas && deploy.Status.ObservedGeneration >= deploy.Generation {
			return true, 100, "Rollout completed: all replicas ready", podStatus, nil
		}

		pct := 50
		if desiredReplicas > 0 {
			pct = 50 + int((float64(readyReplicas)/float64(desiredReplicas))*45)
		}
		msg = fmt.Sprintf("Waiting for rollout: %d/%d replicas ready", readyReplicas, desiredReplicas)
		return false, pct, msg, podStatus, nil

	case "statefulset":
		sts, err := kClient.clientset.AppsV1().StatefulSets(namespace).Get(ctx, workloadName, metav1.GetOptions{})
		if err != nil {
			return false, 0, "", "", fmt.Errorf("failed to get statefulset %s: %w", workloadName, err)
		}

		desiredReplicas := int32(1)
		if sts.Spec.Replicas != nil {
			desiredReplicas = *sts.Spec.Replicas
		}

		updatedReplicas := sts.Status.UpdatedReplicas
		readyReplicas := sts.Status.ReadyReplicas

		if updatedReplicas == desiredReplicas && readyReplicas == desiredReplicas && sts.Status.ObservedGeneration >= sts.Generation {
			return true, 100, "Rollout completed: statefulset replicas ready", podStatus, nil
		}

		pct := 50
		if desiredReplicas > 0 {
			pct = 50 + int((float64(readyReplicas)/float64(desiredReplicas))*45)
		}
		msg = fmt.Sprintf("Waiting for rollout: %d/%d replicas ready", readyReplicas, desiredReplicas)
		return false, pct, msg, podStatus, nil

	case "daemonset":
		ds, err := kClient.clientset.AppsV1().DaemonSets(namespace).Get(ctx, workloadName, metav1.GetOptions{})
		if err != nil {
			return false, 0, "", "", fmt.Errorf("failed to get daemonset %s: %w", workloadName, err)
		}

		desired := ds.Status.DesiredNumberScheduled
		ready := ds.Status.NumberReady
		updated := ds.Status.UpdatedNumberScheduled

		if updated == desired && ready == desired && ds.Status.ObservedGeneration >= ds.Generation {
			return true, 100, "Rollout completed: daemonset pods ready", podStatus, nil
		}

		pct := 50
		if desired > 0 {
			pct = 50 + int((float64(ready)/float64(desired))*45)
		}
		msg = fmt.Sprintf("Waiting for rollout: %d/%d pods ready", ready, desired)
		return false, pct, msg, podStatus, nil

	default:
		return true, 100, "Workload updated", podStatus, nil
	}
}

// GetRolloutTimeout returns the configured timeout duration for rollout watcher
func GetRolloutTimeout() time.Duration {
	if envTimeout := os.Getenv("K8S_ROLLOUT_TIMEOUT"); envTimeout != "" {
		if d, err := time.ParseDuration(envTimeout); err == nil {
			return d
		}
	}
	return 5 * time.Minute
}
