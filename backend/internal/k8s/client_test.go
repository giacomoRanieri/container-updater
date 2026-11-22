package k8s

import "testing"

func TestResolveTrackTag(t *testing.T) {
	tests := []struct {
		name          string
		workloadAnns  map[string]string
		podAnns       map[string]string
		containerName string
		want          string
		wantNil       bool
	}{
		{
			name:          "no annotations returns nil",
			workloadAnns:  map[string]string{"container-updater.enable": "true"},
			podAnns:       nil,
			containerName: "n8n",
			wantNil:       true,
		},
		{
			name:          "whitespace only returns nil",
			workloadAnns:  map[string]string{"container-updater.track-tag": "   "},
			podAnns:       map[string]string{"container-updater.n8n.track-tag": "\t"},
			containerName: "n8n",
			wantNil:       true,
		},
		{
			name:          "workload-level track-tag",
			workloadAnns:  map[string]string{"container-updater.track-tag": " stable "},
			containerName: "n8n",
			want:          "stable",
		},
		{
			name:          "pod template fallback when workload annotation absent",
			workloadAnns:  map[string]string{"container-updater.enable": "true"},
			podAnns:       map[string]string{"container-updater.track-tag": "stable"},
			containerName: "matterbridge",
			want:          "stable",
		},
		{
			name: "per-container annotation overrides workload-level in multi-container pod",
			workloadAnns: map[string]string{
				"container-updater.track-tag":           "stable",
				"container-updater.wireguard.track-tag": "latest",
			},
			containerName: "wireguard",
			want:          "latest",
		},
		{
			name: "second container in multi-container pod inherits workload-level",
			workloadAnns: map[string]string{
				"container-updater.track-tag":           "stable",
				"container-updater.wireguard.track-tag": "latest",
			},
			containerName: "wg-portal",
			want:          "stable",
		},
		{
			name:         "per-container pod annotation overrides workload-level generic",
			workloadAnns: map[string]string{"container-updater.track-tag": "stable"},
			podAnns:      map[string]string{"container-updater.wireguard.track-tag": "latest"},
			containerName: "wireguard",
			want:         "latest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveTrackTag(tt.workloadAnns, tt.podAnns, tt.containerName)
			if tt.wantNil {
				if got != nil {
					t.Errorf("resolveTrackTag() = %q, want nil", *got)
				}
				return
			}
			if got == nil || *got != tt.want {
				t.Errorf("resolveTrackTag() = %v, want %q", got, tt.want)
			}
		})
	}
}
