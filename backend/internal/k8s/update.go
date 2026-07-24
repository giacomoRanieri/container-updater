package k8s

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"container-updater/backend/internal/config"
	"container-updater/backend/internal/git"
	"container-updater/backend/internal/logger"

	"gopkg.in/yaml.v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// UpdateK8sWorkload updates the manifest on disk, and optionally updates the cluster directly
func UpdateK8sWorkload(ctx context.Context, namespace, workloadType, workloadName, containerName, newImage string) error {
	// 1. Determine manifest directories to search
	candidateDirs := []string{}
	if manifestsDir := os.Getenv("K8S_MANIFESTS_DIR"); manifestsDir != "" {
		candidateDirs = append(candidateDirs, manifestsDir)
	}
	if os.Getenv("K8S_WORKSPACE_DIR") != "" {
		candidateDirs = append(candidateDirs, os.Getenv("K8S_WORKSPACE_DIR"))
	}
	if dir := os.Getenv("WORKSPACE_DIR"); dir != "" {
		candidateDirs = append(candidateDirs, dir)
	}
	if len(candidateDirs) == 0 {
		candidateDirs = []string{"/app/manifests", "/app/workspace"}
	}

	// If GitOps is enabled, use the cloned repo directory first
	if config.GlobalConfig.GitOps.Enabled {
		gitCloneDir := os.Getenv("GITOPS_CLONE_DIR")
		if gitCloneDir == "" {
			gitCloneDir = "/app/gitops"
		}
		candidateDirs = append([]string{gitCloneDir}, candidateDirs...)
	}

	logger.Log.Info("searching for workload manifest file...", "dirs", candidateDirs, "name", workloadName, "type", workloadType)

	// 2. Walk directories to find and edit the manifest
	found, err := findAndEditManifestInRoots(candidateDirs, workloadType, workloadName, containerName, newImage)
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
		if found {
			if err := git.ProcessGitOpsCommit(ctx, workloadName, newImage); err != nil {
				return fmt.Errorf("gitops sync failed: %w", err)
			}
		}
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

// findAndEditManifestInRoots recursively searches for a yaml file defining the workload,
// updates the image path, and saves the file back.
func findAndEditManifestInRoots(roots []string, workloadType, workloadName, containerName, newImage string) (bool, error) {
	manifestFound := false

	for _, root := range roots {
		if root == "" {
			continue
		}
		found, err := findAndEditManifest(root, workloadType, workloadName, containerName, newImage)
		if err != nil {
			return false, err
		}
		if found {
			manifestFound = true
			break
		}
	}

	return manifestFound, nil
}

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

		// Decode all documents in multi-document YAML file
		decoder := yaml.NewDecoder(bytes.NewReader(data))
		var docs []map[string]any

		for {
			var doc map[string]any
			if err := decoder.Decode(&doc); err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				break
			}
			if doc != nil {
				docs = append(docs, doc)
			}
		}

		if len(docs) == 0 {
			return nil
		}

		updatedInFile := false

		for _, doc := range docs {
			kind, _ := doc["kind"].(string)
			metadata, _ := doc["metadata"].(map[string]any)
			if metadata == nil {
				continue
			}
			name, _ := metadata["name"].(string)

			// Compare kind and name
			if strings.EqualFold(kind, workloadType) && strings.EqualFold(name, workloadName) {
				logger.Log.Info("found matching manifest YAML file", "path", path)

				spec, _ := doc["spec"].(map[string]any)
				if spec == nil {
					continue
				}
				template, _ := spec["template"].(map[string]any)
				if template == nil {
					continue
				}
				templateSpec, _ := template["spec"].(map[string]any)
				if templateSpec == nil {
					continue
				}
				containers, _ := templateSpec["containers"].([]any)
				if containers == nil {
					continue
				}

				// Update container image
				updatedContainer := false
				for i, cItem := range containers {
					cMap, ok := cItem.(map[string]any)
					if !ok {
						continue
					}
					cName, _ := cMap["name"].(string)
					if strings.EqualFold(cName, containerName) {
						cMap["image"] = newImage
						containers[i] = cMap
						updatedContainer = true
						break
					}
				}

				if updatedContainer {
					templateSpec["containers"] = containers
					template["spec"] = templateSpec
					spec["template"] = template
					doc["spec"] = spec
					updatedInFile = true
				}
			}
		}

		if updatedInFile {
			var buf bytes.Buffer
			encoder := yaml.NewEncoder(&buf)
			encoder.SetIndent(2)

			for _, doc := range docs {
				if err := encoder.Encode(doc); err != nil {
					return err
				}
			}
			encoder.Close()

			if err := os.WriteFile(path, buf.Bytes(), 0644); err != nil {
				return err
			}

			manifestFound = true
			return filepath.SkipDir // Stop walking since we found and edited
		}

		return nil
	})

	if err != nil && !errors.Is(err, filepath.SkipDir) {
		return false, err
	}

	return manifestFound, nil
}
