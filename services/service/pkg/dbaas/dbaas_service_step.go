package dbaas

import (
	v2 "github.com/Netcracker/qubership-cassandra-supplementary/api/v1alpha1"
	"github.com/Netcracker/qubership-cassandra-supplementary/pkg/utils"
	"github.com/Netcracker/qubership-nosqldb-operator-core/pkg/constants"
	"github.com/Netcracker/qubership-nosqldb-operator-core/pkg/core"
	cUtils "github.com/Netcracker/qubership-nosqldb-operator-core/pkg/utils"
	"go.uber.org/zap"
	v12 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type DbaasService struct {
	core.DefaultExecutable
}

func (r *DbaasService) Execute(ctx core.ExecutionContext) error {
	kubeClient := ctx.Get(constants.ContextClient).(client.Client)
	request := ctx.Get(constants.ContextRequest).(reconcile.Request)
	log := ctx.Get(constants.ContextLogger).(*zap.Logger)
	spec := ctx.Get(constants.ContextSpec).(*v2.CassandraSupplService)

	template := cUtils.SimpleServiceTemplate(
		utils.DbaasName,
		map[string]string{
			constants.App:          utils.CassandraCluster,
			constants.Microservice: utils.DbaasName,
			utils.Name:             utils.DbaasName,
		},
		map[string]string{
			utils.Name: utils.DbaasName,
		},
		map[string]int32{"http": utils.GetHTTPPort(utils.IsTLSEnableForDBAAS(spec.Spec.Dbaas.Aggregator.DbaasAggregatorRegistrationAddress, spec.Spec.TLS.Enabled))}, request.Namespace)
	// Kubernetes api causes "invalid resourceVersion error" on update. So remove it.
	core.DeleteRuntimeObject(kubeClient, &v12.Service{
		ObjectMeta: template.ObjectMeta,
	})

	labels := utils.BasicLabels{
		AppName:              utils.DbaasName,
		AppComponent:         "backend",
		AppTechnology:        "go",
		AppPartOf:            "cassandra-services",
		AppManagedBy:         "operator",
		AppManagedByOperator: "cassandra-services-operator",
	}

	err := utils.CreateRuntimeObjectContextWrapper(ctx, template, template.ObjectMeta, labels)
	core.PanicError(err, log.Error, "Dbaas service creation failed")

	log.Debug("Dbaas Service has been created")

	return nil
}
