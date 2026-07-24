package api

import (
	"testing"
)

func TestParseK8sWorkloadID(t *testing.T) {
	tests := []struct {
		name          string
		id            string
		wantNamespace string
		wantWType     string
		wantWName     string
		wantCName     string
		wantErr       bool
	}{
		{
			name:          "Colon format with hyphens in deployment and container name",
			id:            "k8s:default:deploy:matter-server:matter-server",
			wantNamespace: "default",
			wantWType:     "Deployment",
			wantWName:     "matter-server",
			wantCName:     "matter-server",
			wantErr:       false,
		},
		{
			name:          "Colon format for StatefulSet",
			id:            "k8s:kube-system:sts:redis-cluster:redis",
			wantNamespace: "kube-system",
			wantWType:     "StatefulSet",
			wantWName:     "redis-cluster",
			wantCName:     "redis",
			wantErr:       false,
		},
		{
			name:          "Colon format for DaemonSet",
			id:            "k8s:monitoring:ds:node-exporter:node-exporter",
			wantNamespace: "monitoring",
			wantWType:     "DaemonSet",
			wantWName:     "node-exporter",
			wantCName:     "node-exporter",
			wantErr:       false,
		},
		{
			name:          "Legacy hyphen format with identical hyphenated names",
			id:            "k8s-default-deploy-matter-server-matter-server",
			wantNamespace: "default",
			wantWType:     "Deployment",
			wantWName:     "matter-server",
			wantCName:     "matter-server",
			wantErr:       false,
		},
		{
			name:          "Legacy hyphen format simple names",
			id:            "k8s-default-deploy-nginx-nginx",
			wantNamespace: "default",
			wantWType:     "Deployment",
			wantWName:     "nginx",
			wantCName:     "nginx",
			wantErr:       false,
		},
		{
			name:          "Legacy hyphen format with sidecar container",
			id:            "k8s-default-deploy-my-app-sidecar",
			wantNamespace: "default",
			wantWType:     "Deployment",
			wantWName:     "my-app",
			wantCName:     "sidecar",
			wantErr:       false,
		},
		{
			name:    "Invalid prefix",
			id:      "docker-12345",
			wantErr: true,
		},
		{
			name:    "Invalid indicator",
			id:      "k8s:default:unknown:foo:bar",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ns, wType, wName, cName, err := parseK8sWorkloadID(tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseK8sWorkloadID(%q) error = %v, wantErr %v", tt.id, err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if ns != tt.wantNamespace {
					t.Errorf("namespace = %q, want %q", ns, tt.wantNamespace)
				}
				if wType != tt.wantWType {
					t.Errorf("wType = %q, want %q", wType, tt.wantWType)
				}
				if wName != tt.wantWName {
					t.Errorf("wName = %q, want %q", wName, tt.wantWName)
				}
				if cName != tt.wantCName {
					t.Errorf("cName = %q, want %q", cName, tt.wantCName)
				}
			}
		})
	}
}
