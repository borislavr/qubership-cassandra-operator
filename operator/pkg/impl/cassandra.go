package impl

import (
	"github.com/Netcracker/qubership-cassandra-operator/api/v1alpha1"
	"github.com/Netcracker/qubership-cassandra-operator/pkg/impl/cassandra"
	"github.com/Netcracker/qubership-cassandra-operator/pkg/impl/common"
	"github.com/Netcracker/qubership-cassandra-operator/pkg/impl/utils"
	"github.com/Netcracker/qubership-nosqldb-operator-core/pkg/constants"
	"github.com/Netcracker/qubership-nosqldb-operator-core/pkg/core"
	"github.com/Netcracker/qubership-nosqldb-operator-core/pkg/steps"
	"go.uber.org/zap"
	v1core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type CassandraCompound struct {
	core.DefaultCompound
}

type CassandraBuilder struct {
	core.ExecutableBuilder
}

func (r *CassandraBuilder) Build(ctx core.ExecutionContext) core.Executable {

	spec := ctx.Get(constants.ContextSpec).(*v1alpha1.CassandraDeployment)
	request := ctx.Get(constants.ContextRequest).(reconcile.Request)
	log := ctx.Get(constants.ContextLogger).(*zap.Logger)
	client := ctx.Get(constants.ContextClient).(client.Client)

	log.Debug("Cassandra Executable build process is started")
	// It is needed for test proposes. Implementations is changed for module tests
	// TODO: Force key change based on deploy type?
	defaultKubernetesHelper := &core.DefaultKubernetesHelperImpl{
		ForceKey: spec.Spec.StopOnFailedResourceUpdate,
		OwnerKey: true,
		Client:   client,
	}

	ctx.Set(utils.KubernetesHelperImpl, defaultKubernetesHelper)
	cassandraUtilsHelper := &utils.CassandraUtilsImpl{
		KubernetesHelperImpl: defaultKubernetesHelper,
	}
	ctx.Set(utils.CassandraHelperImpl, cassandraUtilsHelper)

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

	// TODO: Cleanup steps?

	var compound core.ExecutableCompound = &CassandraCompound{}

	//compound.AddStep(steps.NewRegisterConsulServiceStep(
	//	utils.Cassandra,
	//	cassandra.CassandraConsulServiceRegistrationCast))

	if spec.Spec.Cassandra.Install {
		_, cassandraSecretErr := core.ReadSecret(client, spec.Spec.Cassandra.SecretName, request.Namespace)
		if cassandraSecretErr != nil {
			// Absence of cassandra secret is ok if cassandra is not installed
			// Ignore context credentials update
			if !errors.IsNotFound(cassandraSecretErr) {
				core.PanicError(cassandraSecretErr, log.Error, "Failed checking cassandra secret")
			}
			log.Warn("Cassandra secret doesn't exist!")
		} else {
			// If secret exists select the strategy to save password to context
			if !spec.Spec.VaultRegistration.Enabled {
				compound.AddStep(&common.SetPasswordFromSecret{})
			} else {
				compound.AddStep(&steps.SetPasswordFromVaultRole{
					Registration:          spec.Spec.VaultRegistration,
					RoleName:              spec.Spec.VaultDBEngine.Role,
					CtxVarToStorePassword: utils.ContextPasswordKey,
				})
			}
		}

		compound.AddStep(&common.InitialValidations{})
		compound.AddStep(&common.SeedsStep{})
		compound.AddStep(&cassandra.CassandraConfigurationUpgradeStep{})
		compound.AddStep((&cassandra.CassandraBuilder{}).Build(ctx))
	}

	compound.AddStep(steps.NewRegisterConsulServiceStep(
		utils.Cassandra,
		cassandra.CassandraConsulServiceRegistrationCast))

	log.Debug("Cassandra Executable has been built")

	return compound
}

type PreDeployCassandraBuilder struct {
	core.ExecutableBuilder
}

func (r *PreDeployCassandraBuilder) Build(ctx core.ExecutionContext) core.Executable {
	var compound core.ExecutableCompound = &CassandraCompound{}
	spec := ctx.Get(constants.ContextSpec).(*v1alpha1.CassandraDeployment)
	client := ctx.Get(constants.ContextClient).(client.Client)
	// request := ctx.Get(constants.ContextRequest).(reconcile.Request)
	// kubeConfig := ctx.Get(constants.ContextKubeClient).(*rest.Config)
	// client := ctx.Get(constants.ContextClient).(v2.Client)

	//TODO DR
	defaultUtilsHelper := &core.DefaultKubernetesHelperImpl{
		ForceKey: spec.Spec.StopOnFailedResourceUpdate,
		OwnerKey: true,
		Client:   client,
	}
	cassandraUtilsHelper := &utils.CassandraUtilsImpl{
		KubernetesHelperImpl: defaultUtilsHelper,
	}

	ctx.Set(utils.KubernetesHelperImpl, defaultUtilsHelper)
	ctx.Set(utils.CassandraHelperImpl, cassandraUtilsHelper)

	if spec.Spec.Cassandra.Install {
		compound.AddStep(&common.SetPasswordFromSecret{})
		compound.AddStep(&common.SeedsStep{})
	}

	if !ctx.Get(constants.ContextSpecHasChanges).(bool) {
		if spec.Spec.VaultRegistration.Enabled {
			compound.AddStep(&steps.SetPasswordFromVaultRole{
				Registration:          spec.Spec.VaultRegistration,
				RoleName:              spec.Spec.VaultDBEngine.Role,
				CtxVarToStorePassword: utils.ContextPasswordKey,
			})
		}
		compound.AddStep(&common.RunFiberServer{})
	}

	// compound.AddStep(&cleanup.DeleteUnusedSecrets{})

	return compound
}
