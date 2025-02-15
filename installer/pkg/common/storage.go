// Copyright (c) 2021 Gitpod GmbH. All rights reserved.
// Licensed under the GNU Affero General Public License (AGPL).
// See License-AGPL.txt in the project root for license information.

package common

import (
	"fmt"

	storageconfig "github.com/gitpod-io/gitpod/content-service/api/config"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"
)

// StorageConfig produces config service configuration from the installer config

func StorageConfig(context *RenderContext) storageconfig.StorageConfig {
	var res *storageconfig.StorageConfig
	if context.Config.ObjectStorage.CloudStorage != nil {
		res = &storageconfig.StorageConfig{
			Kind: storageconfig.GCloudStorage,
			GCloudConfig: storageconfig.GCPConfig{
				Region:             context.Config.Metadata.Region,
				Project:            context.Config.ObjectStorage.CloudStorage.Project,
				CredentialsFile:    "/mnt/secrets/gcp-storage/service-account.json",
				ParallelUpload:     6,
				MaximumBackupCount: 3,
			},
		}
	}
	if context.Config.ObjectStorage.S3 != nil {
		// TODO(cw): where do we get the AWS secretKey and accessKey from?
		res = &storageconfig.StorageConfig{
			Kind: storageconfig.MinIOStorage,
			MinIOConfig: storageconfig.MinIOConfig{
				Endpoint:        "some-magic-amazon-value?",
				AccessKeyID:     "TODO",
				SecretAccessKey: "TODO",
				Secure:          true,
				Region:          context.Config.Metadata.Region,
				ParallelUpload:  6,
			},
		}
	}
	if b := context.Config.ObjectStorage.InCluster; b != nil && *b {
		res = &storageconfig.StorageConfig{
			Kind: storageconfig.MinIOStorage,
			MinIOConfig: storageconfig.MinIOConfig{
				Endpoint:        "minio:9000",
				AccessKeyID:     context.Values.StorageAccessKey,
				SecretAccessKey: context.Values.StorageSecretKey,
				Secure:          false,
				Region:          context.Config.Metadata.Region,
				ParallelUpload:  6,
			},
		}
	}

	if res == nil {
		panic("no valid storage configuration set")
	}

	// todo(sje): create exportable type
	res.BackupTrail = struct {
		Enabled   bool `json:"enabled"`
		MaxLength int  `json:"maxLength"`
	}{
		Enabled:   true,
		MaxLength: 3,
	}
	// 5 GiB
	res.BlobQuota = 5 * 1024 * 1024 * 1024

	return *res
}

// AddStorageMounts adds mounts and volumes to a pod which are required for
// the storage configration to function. If a list of containers is provided,
// the mounts are only added to those container. If the list is empty, they're
// addded to all containers.
func AddStorageMounts(ctx *RenderContext, pod *corev1.PodSpec, container ...string) error {
	if ctx.Config.ObjectStorage.CloudStorage != nil {
		volumeName := "storage-config-cloudstorage"
		pod.Volumes = append(pod.Volumes,
			corev1.Volume{
				Name: volumeName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: ctx.Config.ObjectStorage.CloudStorage.ServiceAccount.Name,
					},
				},
			},
		)

		idx := make(map[string]struct{}, len(container))
		if len(container) == 0 {
			for _, c := range pod.Containers {
				idx[c.Name] = struct{}{}
			}
		} else {
			for _, c := range container {
				idx[c] = struct{}{}
			}
		}

		for i := range pod.Containers {
			if _, ok := idx[pod.Containers[i].Name]; !ok {
				continue
			}

			pod.Containers[i].VolumeMounts = append(pod.Containers[i].VolumeMounts,
				corev1.VolumeMount{
					Name:      volumeName,
					ReadOnly:  true,
					MountPath: "/mnt/secrets/gcp-storage",
				},
			)
		}

		return nil
	}

	if ctx.Config.ObjectStorage.S3 != nil {
		return nil
	}

	if pointer.BoolDeref(ctx.Config.ObjectStorage.InCluster, false) {
		// builtin storage needs no extra mounts
		return nil
	}

	return fmt.Errorf("no valid storage confniguration set")
}
