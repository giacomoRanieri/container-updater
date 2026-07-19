package k8s

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"container-updater/backend/internal/config"
	"container-updater/backend/internal/logger"

	"gopkg.in/yaml.v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// UpdateK8sWorkload updates the manifest on disk, and optionally updates the cluster directly
func UpdateK8sWorkload(ctx context.Context, namespace, workloadType, workloadName, containerName, newImage string) error {
	// 1. Determine manifest directory
	manifestsDir := os.Getenv("K8S_MANIFESTS_DIR")
	if manifestsDir == "" {
		manifestsDir = "/app/manifests"
	}
	
	// If GitOps is enabled, use the cloned repo directory instead
	if config.GlobalConfig.GitOps.Enabled {
		gitCloneDir := os.Getenv("GITOPS_CLONE_DIR")
		if gitCloneDir == "" {
			gitCloneDir = "/app/gitops"
		}
		manifestsDir = gitCloneDir
	}

	logger.Log.Info("searching for workload manifest file...", "dir", manifestsDir, "name", workloadName, "type", workloadType)

	// 2. Walk directory to find and edit the manifest
	found, err := findAndEditManifest(manifestsDir, workloadType, workloadName, containerName, newImage)
	if err != nil {
		return fmt.Errorf("failed to search and edit manifest file: %w", err)
	}

	if !found {
		logger.Log.Warn("workload manifest file not found on disk, proceeding with in-cluster update only", "name", workloadName)
	} else {
		logger.Log.Info("manifest file updated on disk successfully")
	}

	// 3. If GitOps is enabled, skip direct in-cluster update (GitOps controller will reconcile)
	if config.GlobalConfig.GitOps.Enabled {
		logger.Log.Info("GitOps enabled. Skipping direct in-cluster update. Changes must be committed and pushed to Git.")
		return nil
	}

	// 4. If GitOps is disabled, apply changes directly to the Kubernetes cluster
	logger.Log.Info("applying update directly to Kubernetes cluster", "workload", workloadName, "type", workloadType)
	kClient, err := NewK8sClient()
	if err != nil {
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	switch strings.ToLower(workloadType) {
	case "deployment":
		deploy, err := kClient.clientset.AppsV1().Deployments(namespace).Get(ctx, workloadName, metav1.GetOptions{})
		if err != nil {
			return err
		}
		
		updated := false
		for i, c := range deploy.Spec.Template.Spec.Containers {
			if c.Name == containerName {
				deploy.Spec.Template.Spec.Containers[i].Image = newImage
				updated = true
				break
			}
		}
		if !updated {
			return fmt.Errorf("container %s not found in deployment %s", containerName, workloadName)
		}
		_, err = kClient.clientset.AppsV1().Deployments(namespace).Update(ctx, deploy, metav1.UpdateOptions{})
		if err != nil {
			return err
		}

	case "statefulset":
		sts, err := kClient.clientset.AppsV1().StatefulSets(namespace).Get(ctx, workloadName, metav1.GetOptions{})
		if err != nil {
			return err
		}
		
		updated := false
		for i, c := range sts.Spec.Template.Spec.Containers {
			if c.Name == containerName {
				sts.Spec.Template.Spec.Containers[i].Image = newImage
				updated = true
				break
			}
		}
		if !updated {
			return fmt.Errorf("container %s not found in statefulset %s", containerName, workloadName)
		}
		_, err = kClient.clientset.AppsV1().StatefulSets(namespace).Update(ctx, sts, metav1.UpdateOptions{})
		if err != nil {
			return err
		}

	case "daemonset":
		ds, err := kClient.clientset.AppsV1().DaemonSets(namespace).Get(ctx, workloadName, metav1.GetOptions{})
		if err != nil {
			return err
		}
		
		updated := false
		for i, c := range ds.Spec.Template.Spec.Containers {
			if c.Name == containerName {
				ds.Spec.Template.Spec.Containers[i].Image = newImage
				updated = true
				break
			}
		}
		if !updated {
			return fmt.Errorf("container %s not found in daemonset %s", containerName, workloadName)
		}
		_, err = kClient.clientset.AppsV1().DaemonSets(namespace).Update(ctx, ds, metav1.UpdateOptions{})
		if err != nil {
			return err
		}

	default:
		return fmt.Errorf("unsupported workload type for in-cluster update: %s", workloadType)
	}

	logger.Log.Info("applied workload in-cluster update successfully", "workload", workloadName)
	return nil
}

// findAndEditManifest recursively searches for a yaml file defining the workload,
// updates the image path, and saves the file back.
func findAndEditManifest(dir, workloadType, workloadName, containerName, newImage string) (bool, error) {
	manifestFound := false
	
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		
		if d.IsDir() {
			return nil
		}

		// Filter YAML files
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		// Unmarshal to verify if this document contains our workload
		var doc map[string]any
		if err := yaml.Unmarshal(data, &doc); err != nil {
			// Skip unparseable YAML documents (could be multi-document or invalid)
			return nil
		}

		kind, _ := doc["kind"].(string)
		metadata, _ := doc["metadata"].(map[string]any)
		if metadata == nil {
			return nil
		}
		name, _ := metadata["name"].(string)

		// Compare kind and name
		if strings.ToLower(kind) == strings.ToLower(workloadType) && strings.ToLower(name) == strings.ToLower(workloadName) {
			logger.Log.Info("found matching manifest YAML file", "path", path)
			
			spec, _ := doc["spec"].(map[string]any)
			if spec == nil {
				return nil
			}
			template, _ := spec["template"].(map[string]any)
			if template == nil {
				return nil
			}
			templateSpec, _ := template["spec"].(map[string]any)
			if templateSpec == nil {
				return nil
			}
			containers, _ := templateSpec["containers"].([]any)
			if containers == nil {
				return nil
			}

			// Update container image
			updated := false
			for i, cItem := range containers {
				cMap, ok := cItem.(map[string]any)
				if !ok {
					continue
				}
				cName, _ := cMap["name"].(string)
				if strings.ToLower(cName) == strings.ToLower(containerName) {
					cMap["image"] = newImage
					containers[i] = cMap
					updated = true
					break
				}
			}

			if updated {
				templateSpec["containers"] = containers
				template["spec"] = templateSpec
				spec["template"] = template
				doc["spec"] = spec

				// Write back updated YAML structure
				outBytes, err := yaml.Marshal(doc)
				if err != nil {
					return err
				}

				if err := os.WriteFile(path, outBytes, 0644); err != nil {
					return err
				}

				manifestFound = true
				return filepath.SkipDir // Stop walking since we found and edited
			}
		}

		return nil
	})

	if err != nil && !errors.Is(err, filepath.SkipDir) {
		return false, err
	}

	return manifestFound, nil
}
