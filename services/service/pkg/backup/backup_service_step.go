package backup

import (
	v1 "github.com/Netcracker/qubership-cassandra-supplementary/api/v1alpha1"
	"github.com/Netcracker/qubership-cassandra-supplementary/pkg/utils"
	"github.com/Netcracker/qubership-nosqldb-operator-core/pkg/constants"
	"github.com/Netcracker/qubership-nosqldb-operator-core/pkg/core"
	cUtils "github.com/Netcracker/qubership-nosqldb-operator-core/pkg/utils"
	"go.uber.org/zap"
	v12 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type BackupService struct {
	core.DefaultExecutable
}

func (r *BackupService) Execute(ctx core.ExecutionContext) error {
	client := ctx.Get(constants.ContextClient).(client.Client)
	request := ctx.Get(constants.ContextRequest).(reconcile.Request)
	log := ctx.Get(constants.ContextLogger).(*zap.Logger)
	spec := ctx.Get(constants.ContextSpec).(*v1.CassandraSupplService)

	template := cUtils.SimpleServiceTemplate(
		utils.BackupDaemon,
		map[string]string{
			constants.App:          utils.CassandraCluster,
			constants.Microservice: utils.BackupDaemon,
			utils.Name:             utils.BackupDaemon,
		},
		map[string]string{
			utils.Name: utils.BackupDaemon,
		},
		map[string]int32{"http": utils.GetHTTPPort(spec.Spec.TLS.Enabled)},
		request.Namespace)

	// Kubernetes api causes "invalid resourceVersion error" on update. So remove it.
	core.DeleteRuntimeObject(client, &v12.Service{
		ObjectMeta: template.ObjectMeta,
	})

	labels := utils.BasicLabels{
		AppName:              utils.BackupDaemon,
		AppComponent:         "backend",
		AppTechnology:        "python",
		AppPartOf:            "cassandra-services",
		AppManagedBy:         "operator",
		AppManagedByOperator: "cassandra-services-operator",
	}
	err := utils.CreateRuntimeObjectContextWrapper(ctx, template, template.ObjectMeta, labels)
	core.PanicError(err, log.Error, "Backup service creation failed")

	log.Debug("Backup Service has been created")

	return nil
}
