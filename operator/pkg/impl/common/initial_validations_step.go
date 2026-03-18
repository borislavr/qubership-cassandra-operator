package common

import (
	"fmt"

	"github.com/Netcracker/qubership-cassandra-operator/api/v1alpha1"
	"github.com/Netcracker/qubership-cassandra-operator/pkg/impl/utils"
	"github.com/Netcracker/qubership-nosqldb-operator-core/pkg/constants"
	"github.com/Netcracker/qubership-nosqldb-operator-core/pkg/core"
)

type InitialValidations struct {
	core.Executable
}

func (r *InitialValidations) Validate(ctx core.ExecutionContext) error {
	var spec *v1alpha1.CassandraDeployment = ctx.Get(constants.ContextSpec).(*v1alpha1.CassandraDeployment)

	dcs := utils.FilterDC(spec.Spec.Cassandra.DeploymentSchema.DataCenters, func(dc *v1alpha1.DataCenter) bool { return dc.Deploy })

	if len(dcs) < 1 {
		return &core.ExecutionError{Msg: "Datacenters array should be at least 1"}
	}

	for _, dc := range dcs {
		seedNumber := dc.Seeds
		if dc.SeedList != nil {
			seedNumber = len(dc.SeedList)
		}
		if dc.GetActiveReplicasLen() < 1 {
			return &core.ExecutionError{Msg: fmt.Sprintf("Replicas count of '%s' dc is lower than 1", dc.Name)}
		}
		if seedNumber < 1 {
			return &core.ExecutionError{Msg: fmt.Sprintf("Seeds count of '%s' dc is lower than 1 or more than replicas count. (seeds: %v, replicas: %v)", dc.Name, seedNumber, dc.GetActiveReplicasLen())}
		}
		if dc.Storage[0].MountSettings == nil ||
			dc.Storage[0].MountSettings.Name != "data" ||
			dc.Storage[0].MountSettings.MountPath != "/var/lib/cassandra/data" {
			return &core.ExecutionError{Msg: fmt.Sprintf("%s datacenter's first storage element mount settings are overridden", dc.Name)}
		}
		for ifirst, first := range dc.Storage {
			if first.MountSettings == nil ||
				first.MountSettings.MountPath == "" ||
				first.MountSettings.Name == "" {
				return &core.ExecutionError{Msg: fmt.Sprintf("Check %s datacenter's storage mount settings. Name or MountPath is empty", dc.Name)}
			}
			for isecond, second := range dc.Storage {
				if ifirst != isecond {
					if second.MountSettings != nil {
						if first.MountSettings.MountPath == second.MountSettings.MountPath ||
							first.MountSettings.Name == second.MountSettings.Name {
							return &core.ExecutionError{Msg: fmt.Sprintf("%s datacenter's storage elements mount settings are overlapping", dc.Name)}
						}
					}
				}
			}
		}
	}

	return nil
}

func (r *InitialValidations) Execute(ctx core.ExecutionContext) error {

	return nil
}

func (r *InitialValidations) Condition(ctx core.ExecutionContext) (bool, error) {
	return true, nil
}
