package cassandra

import (
	"github.com/Netcracker/qubership-cassandra-operator/api/v1alpha1"
	"github.com/Netcracker/qubership-cassandra-operator/pkg/impl/utils"
	"github.com/Netcracker/qubership-nosqldb-operator-core/pkg/constants"
	"github.com/Netcracker/qubership-nosqldb-operator-core/pkg/core"
	cUtils "github.com/Netcracker/qubership-nosqldb-operator-core/pkg/utils"
	"go.uber.org/zap"
	v12 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type CassandraLoadbalancerService struct {
	core.DefaultExecutable
}

func (r *CassandraLoadbalancerService) Execute(ctx core.ExecutionContext) error {
	client := ctx.Get(constants.ContextClient).(client.Client)
	request := ctx.Get(constants.ContextRequest).(reconcile.Request)
	log := ctx.Get(constants.ContextLogger).(*zap.Logger)
	spec := ctx.Get(constants.ContextSpec).(*v1alpha1.CassandraDeployment)

	var templates []*v12.Service

	service := cUtils.SimpleServiceTemplate(
		utils.CassandraLb,
		map[string]string{
			constants.App:              utils.CassandraCluster,
			constants.Microservice:     utils.CassandraLb,
			utils.Name:                 utils.Cassandra,
			utils.AppName:              utils.Cassandra,
			utils.AppTechnology:        "go",
			utils.AppComponent:         "backend",
			utils.AppManagedBy:         "operator",
			utils.AppInstance:          spec.Spec.Instance,
			utils.AppManagedByOperator: "cassandra-operator",
		},
		map[string]string{
			utils.Service: utils.CassandraCluster,
		},
		map[string]int32{
			"cql-port":     9042,
			"tcp-upd-port": 8778,
			"reaper":       int32(spec.Spec.Reaper.Port),
		},
		request.Namespace)

	service.Spec.SessionAffinity = "ClientIP"
	templates = append(templates, service)

	for _, template := range templates {
		// Kubernetes api causes "invalid resourceVersion error" on update. So remove it.
		core.DeleteRuntimeObject(client, &v12.Service{
			ObjectMeta: template.ObjectMeta,
		})

		err := utils.CreateRuntimeObjectContextWrapper(ctx, template, template.ObjectMeta)
		core.PanicError(err, log.Error, "Service "+template.Name+" creation failed")

		log.Debug("Service " + template.Name + " has been created")
	}

	log.Debug("Cassandra Loadbalancer Service have been created")

	return nil
}
