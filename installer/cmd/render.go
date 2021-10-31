// Copyright (c) 2021 Gitpod GmbH. All rights reserved.
// Licensed under the GNU Affero General Public License (AGPL).
// See License-AGPL.txt in the project root for license information.

package cmd

import (
	"fmt"
	"github.com/gitpod-io/gitpod/installer/pkg/render"
	"os"
	"strings"

	_ "embed"

	"github.com/gitpod-io/gitpod/installer/pkg/config"
	configv1 "github.com/gitpod-io/gitpod/installer/pkg/config/v1"
	"github.com/gitpod-io/gitpod/installer/pkg/config/versions"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"
)

var renderOpts struct {
	ConfigFN               string
	Namespace              string
	ValidateConfigDisabled bool
}

//go:embed versions.yaml
var versionManifest []byte

// renderCmd represents the render command
var renderCmd = &cobra.Command{
	Use:   "render",
	Short: "Renders the Kubernetes manifests required to install Gitpod",
	Long: `Renders the Kubernetes manifests required to install Gitpod

A config file is required which can be generated with the init command.`,
	Example: `  # Default install.
  gitpod-installer render --config config.yaml | kubectl apply -f -

  # Install Gitpod into a non-default namespace.
  gitpod-installer render --config config.yaml --namespace gitpod | kubectl apply -f -`,
	RunE: func(cmd *cobra.Command, args []string) error {
		_, cfgVersion, cfg, err := loadConfig(renderOpts.ConfigFN)
		if err != nil {
			return err
		}

		var versionMF versions.Manifest
		err = yaml.Unmarshal(versionManifest, &versionMF)
		if err != nil {
			return err
		}

		if !renderOpts.ValidateConfigDisabled {
			if err = runConfigValidation(cfgVersion, cfg); err != nil {
				return err
			}
		}

		objs, err := render.ToYaml(*cfg, versionMF, renderOpts.Namespace)
		if err != nil {
			return err
		}

		fmt.Printf("---\n%s\n", strings.Join(objs, "---\n"))

		return nil
	},
}

func loadConfig(cfgFN string) (rawCfg interface{}, cfgVersion string, cfg *configv1.Config, err error) {
	rawCfg, cfgVersion, err = config.Load(cfgFN)
	if err != nil {
		err = fmt.Errorf("error loading config: %w", err)
		return
	}
	if cfgVersion != config.CurrentVersion {
		err = fmt.Errorf("config version is mismatch: expected %s, got %s", config.CurrentVersion, cfgVersion)
		return
	}
	cfg = rawCfg.(*configv1.Config)

	return rawCfg, cfgVersion, cfg, err
}

func init() {
	rootCmd.AddCommand(renderCmd)

	renderCmd.PersistentFlags().StringVarP(&renderOpts.ConfigFN, "config", "c", os.Getenv("GITPOD_INSTALLER_CONFIG"), "path to the config file")
	renderCmd.PersistentFlags().StringVarP(&renderOpts.Namespace, "namespace", "n", "default", "namespace to deploy to")
	renderCmd.Flags().BoolVar(&renderOpts.ValidateConfigDisabled, "no-validation", false, "if set, the config will not be validated before running")
}
