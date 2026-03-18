package common

import (
	"fmt"
	"strings"

	"github.com/Netcracker/qubership-cassandra-operator/api/v1alpha1"
	"github.com/Netcracker/qubership-cassandra-operator/pkg/impl/utils"
	"github.com/Netcracker/qubership-nosqldb-operator-core/pkg/constants"
	"github.com/Netcracker/qubership-nosqldb-operator-core/pkg/core"
	"go.uber.org/zap"
)

type SeedsStep struct {
	core.DefaultExecutable
}

func (r *SeedsStep) Execute(ctx core.ExecutionContext) error {
	spec := ctx.Get(constants.ContextSpec).(*v1alpha1.CassandraDeployment)
	log := ctx.Get(constants.ContextLogger).(*zap.Logger)

	log.Info("Cassandra Seed creation started")

	dcs := spec.Spec.Cassandra.DeploymentSchema.DataCenters

	result := []string{}

	for dc := 0; dc < len(dcs); dc++ {
		var dcSeed []string
		if dcs[dc].SeedList == nil {
			for ind, replica := range dcs[dc].GetActiveReplicas() {
				// cassandra0-0.cassandra-dc0.cassandra.svc.cluster.local
				//OR
				// cassandra0-0.cassandra.cassandra.svc.cluster.local
				if ind >= dcs[dc].Seeds {
					break
				}

				seed := utils.CalcReplicaHostName(true, dcs, dc, replica, "$(NAMESPACE)")

				dcSeed = append(dcSeed, seed)
			}
		} else {
			if dcs[dc].Deploy {
				dcSeed = dcs[dc].SeedList
			}
		}

		dcSeedString := strings.Join(dcSeed, ", ")
		log.Debug("Result Data Center seed is: " + dcSeedString)

		ctx.Set(fmt.Sprintf(utils.CassandraDCSeedsFormat, dc), dcSeedString)

		result = append(result, dcSeed...)
	}

	seeds := strings.Join(result, ", ")

	log.Debug("Result seed is: " + seeds)

	ctx.Set(utils.CassandraSeeds, seeds)

	return nil
}
