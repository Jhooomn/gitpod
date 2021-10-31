// Copyright (c) 2021 Gitpod GmbH. All rights reserved.
// Licensed under the GNU Affero General Public License (AGPL).
// See License-AGPL.txt in the project root for license information.

package render

import (
	"fmt"
	"github.com/gitpod-io/gitpod/installer/pkg/common"
	"github.com/gitpod-io/gitpod/installer/pkg/components"
	"github.com/gitpod-io/gitpod/installer/pkg/config/v1"
	"github.com/gitpod-io/gitpod/installer/pkg/config/versions"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/yaml"
	"strings"
)

type kubernetesObj struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
}

func generateConfigMap(ctx *common.RenderContext, objs []string) (*string, error) {
	var cfgMapData []string
	component := "gitpod-app"

	cfgMap := corev1.ConfigMap{
		TypeMeta: common.TypeMetaConfigmap,
		ObjectMeta: metav1.ObjectMeta{
			Name:      component,
			Namespace: ctx.Namespace,
			Labels:    common.DefaultLabels(component),
		},
	}

	// Add in the ConfigMap
	marshal, err := yaml.Marshal(cfgMap)
	if err != nil {
		return nil, err
	}

	cfgMapData = append(cfgMapData, string(marshal))

	// Convert to a simplified object that allows us to access the objects
	for _, o := range objs {
		var data kubernetesObj
		err = yaml.Unmarshal([]byte(o), &data)
		if err != nil {
			return nil, err
		}

		marshal, err = yaml.Marshal(data)
		if err != nil {
			return nil, err
		}

		cfgMapData = append(cfgMapData, string(marshal))
	}

	cfgMap.Data = map[string]string{
		"config.yaml": strings.Join(cfgMapData, "---\n"),
	}

	cfgMapYaml, err := yaml.Marshal(cfgMap)
	if err != nil {
		return nil, err
	}

	return pointer.String(string(cfgMapYaml)), nil
}

func ToYaml(cfg config.Config, versionManifest versions.Manifest, namespace string) ([]string, error) {
	out := make([]string, 0)
	ctx, err := common.NewRenderContext(cfg, versionManifest, namespace)
	if err != nil {
		return nil, err
	}

	var k8sObjs common.RenderFunc
	var helmCharts common.HelmFunc
	switch cfg.Kind {
	case config.InstallationFull:
		k8sObjs = components.FullObjects
		helmCharts = components.FullHelmDependencies
	case config.InstallationMeta:
		k8sObjs = components.MetaObjects
		helmCharts = components.MetaHelmDependencies
	case config.InstallationWorkspace:
		k8sObjs = components.WorkspaceObjects
		helmCharts = components.WorkspaceHelmDependencies
	default:
		return nil, fmt.Errorf("unsupported installation kind: %s", cfg.Kind)
	}

	objs, err := common.CompositeRenderFunc(k8sObjs, components.CommonObjects)(ctx)
	if err != nil {
		return nil, err
	}

	charts, err := common.CompositeHelmFunc(helmCharts, components.CommonHelmDependencies)(ctx)
	if err != nil {
		return nil, err
	}

	for _, o := range objs {
		fc, err := yaml.Marshal(o)
		if err != nil {
			return nil, err
		}

		out = append(out, string(fc))
	}

	for _, c := range charts {
		out = append(out, c)
	}

	cfgMap, err := generateConfigMap(ctx, out)
	if err != nil {
		return nil, err
	}

	out = append(out, *cfgMap)

	return out, nil
}
