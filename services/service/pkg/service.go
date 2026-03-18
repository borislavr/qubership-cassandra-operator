package pkg

import (
	"github.com/Netcracker/qubership-cql-driver"
	v1 "github.com/Netcracker/qubership-cassandra-supplementary/api/v1alpha1"
	"github.com/Netcracker/qubership-cassandra-supplementary/pkg/backup"
	"github.com/Netcracker/qubership-cassandra-supplementary/pkg/dbaas"
	"github.com/Netcracker/qubership-cassandra-supplementary/pkg/robotTests"
	"github.com/Netcracker/qubership-cassandra-supplementary/pkg/utils"
	"github.com/Netcracker/qubership-nosqldb-operator-core/pkg/constants"
	"github.com/Netcracker/qubership-nosqldb-operator-core/pkg/core"
	"go.uber.org/zap"
	v1core "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type CassandraServicesCompound struct {
	core.DefaultCompound
}

type CassandraServiceBuilder struct {
	core.ExecutableBuilder
}

func (r *CassandraServiceBuilder) Build(ctx core.ExecutionContext) core.Executable {

	spec := ctx.Get(constants.ContextSpec).(*v1.CassandraSupplService)
	log := ctx.Get(constants.ContextLogger).(*zap.Logger)
	client := ctx.Get(constants.ContextClient).(client.Client)

	log.Debug("Cassandra Executable build process is started")
	defaultKubernetesHelper := &core.DefaultKubernetesHelperImpl{
		ForceKey: spec.Spec.StopOnFailedResourceUpdate,
		OwnerKey: false,
		Client:   client,
	}

	ctx.Set(utils.KubernetesHelperImpl, defaultKubernetesHelper)
	ctx.Set(utils.ContextClusterBuilder, &cql.ClusterBuilderImpl{})
	ctx.Set(utils.ContextCredsManager, &utils.CredsManager{})

	// Default tries for wait or init operations (e.g. hosts unreachable)
	ctx.Set(utils.TriesCount, 5)
	ctx.Set(utils.RetryTimeoutSec, 10)

	var depth int = 1
	names := make(map[string]interface{})
	core.GetFieldsAndNamesByTag(names, "true", "common", spec.Spec, &depth)
	isAnyParamChanged, commonParamCheckErr := core.HasSpecChanged(
		ctx, func(cfgTemplate *v1core.ConfigMap) bool {
			resultCheck := false
			for specKey, specToCheck := range names {
				specHasChanges := core.CompareSpecToCM(ctx, cfgTemplate, specToCheck, specKey)

				if specHasChanges {
					resultCheck = true
				}
			}
			return resultCheck
		},
	)
	core.PanicError(commonParamCheckErr, log.Error, "Error happened during checking common parameters for changes")
	ctx.Set(constants.IsAnyCommonParameterChanged, isAnyParamChanged)

	var compound core.ExecutableCompound = &CassandraServicesCompound{}

	if spec.Spec.Backup.Install {
		if spec.Spec.Backup.LegacyMode {
			compound.AddStep((&backup.BackupBuilder{}).Build(ctx))
		} else {
			compound.AddStep((&backup.BackupBuilder{}).Build(ctx))
		}
	}

	if spec.Spec.Dbaas.Install {
		compound.AddStep((&dbaas.DbaasBuilder{}).Build(ctx))
	}

	if spec.Spec.RobotTests.Install {
		compound.AddStep((&robotTests.RobotBuilder{}).Build(ctx))
	}
	log.Debug("Cassandra Executable has been built")

	return compound
}

type PreDeployBuilder struct {
	core.ExecutableBuilder
}

func (r *PreDeployBuilder) Build(ctx core.ExecutionContext) core.Executable {
	var compound core.ExecutableCompound = &CassandraServicesCompound{}
	spec := ctx.Get(constants.ContextSpec).(*v1.CassandraSupplService)
	client := ctx.Get(constants.ContextClient).(client.Client)

	defaultUtilsHelper := &core.DefaultKubernetesHelperImpl{
		ForceKey: spec.Spec.StopOnFailedResourceUpdate,
		OwnerKey: false,
		Client:   client,
	}
	ctx.Set(utils.KubernetesHelperImpl, defaultUtilsHelper)

	if !ctx.Get(constants.ContextSpecHasChanges).(bool) {
		if spec.Spec.VaultRegistration.Enabled {
			// TODO vault
			// compound.AddStep(&steps.SetPasswordFromVaultRole{
			// 	Registration:          spec.Spec.VaultRegistration,
			// 	RoleName:              spec.Spec.VaultDBEngine.Role,
			// 	CtxVarToStorePassword: utils.ContextPasswordKey,
			// })
		}
	}

	return compound
}
