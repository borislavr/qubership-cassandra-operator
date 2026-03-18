package backup

import (
	"fmt"

	v1 "github.com/Netcracker/qubership-cassandra-supplementary/api/v1alpha1"
	"github.com/Netcracker/qubership-cassandra-supplementary/pkg/utils"
	"github.com/Netcracker/qubership-nosqldb-operator-core/pkg/constants"
	"github.com/Netcracker/qubership-nosqldb-operator-core/pkg/core"
	"github.com/Netcracker/qubership-nosqldb-operator-core/pkg/steps"
	"go.uber.org/zap"
	v12 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type CassandraBackup struct {
	core.MicroServiceCompound
}

type BackupBuilder struct {
	core.ExecutableBuilder
}

func (r *BackupBuilder) Build(ctx core.ExecutionContext) core.Executable {
	spec := ctx.Get(constants.ContextSpec).(*v1.CassandraSupplService)

	backupSpec := spec.Spec.Backup
	storage := backupSpec.Storage

	pvcSelector := map[string]string{
		utils.Name: utils.BackupDaemon,
	}

	backup := CassandraBackup{}
	backup.ServiceName = utils.Backup
	backup.CalcDeployType = func(ctx core.ExecutionContext) (core.MicroServiceDeployType, error) {
		request := ctx.Get(constants.ContextRequest).(reconcile.Request)
		log := ctx.Get(constants.ContextLogger).(*zap.Logger)
		helperImpl := ctx.Get(utils.KubernetesHelperImpl).(core.KubernetesHelper)

		pvcList := &v12.PersistentVolumeClaimList{}
		err := helperImpl.ListRuntimeObjectsByLabels(pvcList, request.Namespace, pvcSelector)
		var result core.MicroServiceDeployType
		if err != nil {
			result = core.Empty
		} else if len(pvcList.Items) == 0 {
			result = core.CleanDeploy
		} else {
			result = core.Update
		}

		if err == nil {
			log.Debug(fmt.Sprintf("%s deploy mode is used for %s service", result, utils.Backup))
		}

		return result, err
	}

	pvcContext := fmt.Sprintf(utils.BackupPvcName, 0)
	nodesContext := fmt.Sprintf(utils.PVNodesFormat, 0)

	if !spec.Spec.Backup.Storage.EmptyDir {
		pvcStep := &steps.CreatePVCStep{
			Storage:           storage,
			NameFormat:        utils.BackupPvcName,
			LabelSelector:     pvcSelector,
			ContextVarToStore: pvcContext,
			PVCCount: func(ctx core.ExecutionContext) int {
				return 1
			},
			WaitTimeout:  spec.Spec.WaitTimeout,
			Owner:        nil,
			WaitPVCBound: spec.Spec.Backup.Storage.WaitPVCBound,
		}

		if spec.Spec.DeletePVConUninstall {
			pvcStep.Owner = spec
		}

		backup.AddStep(pvcStep)
		backup.AddStep(&steps.StoreNodesStep{
			Storage:           storage,
			ContextVarToStore: nodesContext,
		})
	}

	backup.AddStep(&BackupService{})

	if spec.Spec.VaultRegistration.Enabled {
		backup.AddStep(&steps.MoveSecretToVault{
			SecretName:        spec.Spec.Backup.SecretName,
			PolicyName:        utils.Backup,
			Policy:            fmt.Sprintf("length = 10\nrule \"charset\" {\n  charset = \"%s\"\n}\n", utils.Charset),
			VaultRegistration: &spec.Spec.VaultRegistration,
		})
	}

	if !spec.Spec.AWSKeyspaces.Install {
		backup.AddStep(&BackupSSHKeyStep{})
	}

	backup.AddStep(&LegacyBackupDeployment{})

	return &backup
}

func (r *CassandraBackup) Condition(ctx core.ExecutionContext) (bool, error) {
	spec := ctx.Get(constants.ContextSpec).(*v1.CassandraSupplService)
	microServiceCheck, microserviceCheckErr := core.CheckSpecChange(ctx, spec.Spec.Backup, utils.BackupDaemon)
	commonCheck := ctx.Get(constants.IsAnyCommonParameterChanged).(bool)

	if microserviceCheckErr != nil {
		return microServiceCheck, microserviceCheckErr
	} else {
		return microServiceCheck || commonCheck, nil
	}
}
