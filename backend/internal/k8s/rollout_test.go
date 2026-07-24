package k8s

import (
	"context"
	"testing"
	"time"
)

func TestGetRolloutTimeout(t *testing.T) {
	t.Setenv("K8S_ROLLOUT_TIMEOUT", "2m")
	timeout := GetRolloutTimeout()
	if timeout != 2*time.Minute {
		t.Errorf("GetRolloutTimeout() = %v, want 2m", timeout)
	}

	t.Setenv("K8S_ROLLOUT_TIMEOUT", "")
	timeoutDefault := GetRolloutTimeout()
	if timeoutDefault != 5*time.Minute {
		t.Errorf("GetRolloutTimeout() default = %v, want 5m", timeoutDefault)
	}
}

func TestWatchRollout_NoK8sClient(t *testing.T) {
	// Out of cluster without kubeconfig should gracefully log warning and return nil
	called := false
	err := WatchRollout(context.Background(), "default", "deployment", "foo", 50*time.Millisecond, func(pct int, msg string, podStatus string) {
		called = true
	})
	if err != nil {
		t.Errorf("WatchRollout without k8s client returned error: %v", err)
	}
	if !called {
		t.Errorf("WatchRollout should invoke callback even if k8s client is unavailable")
	}
}
