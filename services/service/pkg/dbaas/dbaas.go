package dbaas

import (
	"fmt"

	v1 "github.com/Netcracker/qubership-cassandra-supplementary/api/v1alpha1"
	"github.com/Netcracker/qubership-cassandra-supplementary/pkg/utils"
	"github.com/Netcracker/qubership-nosqldb-operator-core/pkg/constants"
	"github.com/Netcracker/qubership-nosqldb-operator-core/pkg/core"
	"github.com/Netcracker/qubership-nosqldb-operator-core/pkg/steps"
)

type DbaasCompound struct {
	core.MicroServiceCompound
}

type DbaasBuilder struct {
	core.ExecutableBuilder
}

func (r *DbaasBuilder) Build(ctx core.ExecutionContext) core.Executable {
	spec := ctx.Get(constants.ContextSpec).(*v1.CassandraSupplService)
	dbaas := DbaasCompound{}
	dbaas.ServiceName = utils.Dbaas
	dbaas.CalcDeployType = func(ctx core.ExecutionContext) (deployType core.MicroServiceDeployType, err error) {
		return core.CleanDeploy, nil
	}

	dbaas.AddStep(&DbaasService{})

	if spec.Spec.VaultRegistration.Enabled {
		dbaas.AddStep(&steps.MoveSecretToVault{
			SecretName:        spec.Spec.Dbaas.Adapter.SecretName,
			PolicyName:        utils.Dbaas,
			Policy:            fmt.Sprintf("length = 10\nrule \"charset\" {\n  charset = \"%s\"\n}\n", utils.Charset),
			VaultRegistration: &spec.Spec.VaultRegistration,
		})
	}

	dbaas.AddStep(&DbaasDeployment{})

	return &dbaas
}

func (r *DbaasCompound) Condition(ctx core.ExecutionContext) (bool, error) {
	spec := ctx.Get(constants.ContextSpec).(*v1.CassandraSupplService)
	microServiceCheck, microserviceCheckErr := core.CheckSpecChange(ctx, spec.Spec.Dbaas, utils.DbaasName)
	commonCheck := ctx.Get(constants.IsAnyCommonParameterChanged).(bool)

	if microserviceCheckErr != nil {
		return microServiceCheck, microserviceCheckErr
	} else {
		return microServiceCheck || commonCheck, nil
	}
}
