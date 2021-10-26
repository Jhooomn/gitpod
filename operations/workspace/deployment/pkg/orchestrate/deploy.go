// Copyright (c) 2021 Gitpod GmbH. All rights reserved.
// Licensed under the GNU Affero General Public License (AGPL).
// See License-AGPL.txt in the project root for license information.

package orchestrate

import (
	"sync"

	log "github.com/gitpod-io/gitpod/common-go/log"
	"github.com/gitpod-io/gitpod/ws-deployment/pkg/common"
	"github.com/gitpod-io/gitpod/ws-deployment/pkg/step"
)

type clusterErrorPair struct {
	cluster common.WorkspaceCluster
	err     error
}

func Deploy(context *common.ProjectContext, gitpodContext common.GitpodContext, clusters []*common.WorkspaceCluster) error {
	var wg sync.WaitGroup
	wg.Add(len(clusters))
	pairChannel := make(chan clusterErrorPair, len(clusters))
	createClusters(pairChannel, context, clusters)

	wg.Wait()
	close(pairChannel)
	return nil
}

func installGitpodOnClusters(pairChannelInput chan clusterErrorPair, pairChannelOutput chan clusterErrorPair, cluster common.WorkspaceCluster, context *common.ProjectContext, gitpodContext common.GitpodContext) {
	for pairChannelInput != nil {
		select {
		case pair, ok := <-pairChannelInput:
			if !ok {
				pairChannelInput = nil
			}
			go installGitpodOnCluster(pairChannelOutput, pair.cluster, context, gitpodContext)
		}
		if pairChannelInput == nil {
			break
		}
	}

}
func installGitpodOnCluster(pairChannelOutput chan clusterErrorPair, cluster common.WorkspaceCluster, context *common.ProjectContext, gitpodContext common.GitpodContext) {
	err := step.InstallGitpod(context, &gitpodContext, &cluster)
	if err != nil {
		log.Log.Infof("error installing gitpod on cluster %s: %s", cluster.Name, err)
	}
	pairChannelOutput <- clusterErrorPair{cluster: cluster, err: err}
}

func createClusters(pairChannel chan clusterErrorPair, context *common.ProjectContext, clusters []*common.WorkspaceCluster) {
	for _, cluster := range clusters {
		go func(context *common.ProjectContext, cluster *common.WorkspaceCluster, pair chan clusterErrorPair) {
			// TODO(prs): add retry logic below
			err := step.CreateCluster(context, cluster)
			if err != nil {
				log.Log.Infof("error creating cluster %s: %s", cluster.Name, err)
			}
			pairChannel <- clusterErrorPair{cluster: *cluster, err: err}
		}(context, cluster, pairChannel)
	}
}
