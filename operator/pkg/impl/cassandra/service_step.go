package cassandra

import (
	"fmt"

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

type CassandraServicesStep struct {
	core.DefaultExecutable
}

func (r *CassandraServicesStep) Execute(ctx core.ExecutionContext) error {
	client := ctx.Get(constants.ContextClient).(client.Client)
	request := ctx.Get(constants.ContextRequest).(reconcile.Request)
	spec := ctx.Get(constants.ContextSpec).(*v1alpha1.CassandraDeployment)
	log := ctx.Get(constants.ContextLogger).(*zap.Logger)

	var templates []*v12.Service

	headless := cUtils.HeadlessServiceTemplate(
		utils.Cassandra,
		map[string]string{
			constants.App:              utils.CassandraCluster,
			constants.Microservice:     utils.Cassandra,
			utils.Name:                 utils.Cassandra,
			utils.AppName:              utils.Cassandra,
			utils.AppTechnology:        "java",
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

	headless.Spec.SessionAffinity = "ClientIP"

	// TODO: Active-Active (Selectors)
	templates = append(templates, headless)

	//TODO monitoring switch
	templates = append(templates, cUtils.SimpleServiceTemplate(
		utils.CassandraMetrics,
		map[string]string{
			constants.App:              utils.CassandraCluster,
			constants.Microservice:     utils.CassandraMetrics,
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
			"metrics": 9500,
		},
		request.Namespace))

	for _, dc := range utils.FilterDC(spec.Spec.Cassandra.DeploymentSchema.DataCenters, func(dc *v1alpha1.DataCenter) bool { return dc.Deploy }) {
		dcLabel := fmt.Sprintf(utils.CassandraDCFormat, dc.Name)

		templates = append(templates, cUtils.HeadlessServiceTemplate(
			dcLabel,
			map[string]string{
				constants.App:              utils.CassandraCluster,
				constants.Microservice:     dcLabel,
				utils.Name:                 utils.Cassandra,
				utils.AppName:              utils.Cassandra,
				utils.AppTechnology:        "go",
				utils.AppComponent:         "backend",
				utils.AppManagedBy:         "operator",
				utils.AppInstance:          spec.Spec.Instance,
				utils.AppManagedByOperator: "cassandra-operator",
			},
			map[string]string{
				utils.App: dcLabel,
			},
			map[string]int32{
				"cql-port": 9042,
			},
			request.Namespace))
	}

	for _, template := range templates {
		// Kubernetes api causes "invalid resourceVersion error" on update. So remove it.
		core.DeleteRuntimeObject(client, &v12.Service{
			ObjectMeta: template.ObjectMeta,
		})

		err := utils.CreateRuntimeObjectContextWrapper(ctx, template, template.ObjectMeta)
		core.PanicError(err, log.Error, "Service "+template.Name+" creation failed")

		log.Debug("Service " + template.Name + " has been created")
	}

	log.Debug("Cassandra Services have been created")

	return nil
}
