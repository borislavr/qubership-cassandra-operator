package cassandra

import (
	"fmt"
	"strconv"

	"github.com/Netcracker/qubership-cassandra-operator/api/v1alpha1"
	"github.com/Netcracker/qubership-cassandra-operator/pkg/impl/utils"
	"github.com/Netcracker/qubership-nosqldb-operator-core/pkg/constants"
	"github.com/Netcracker/qubership-nosqldb-operator-core/pkg/core"
	"go.uber.org/zap"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type CleanupNodes struct {
	core.DefaultExecutable
}

func (r *CleanupNodes) Execute(ctx core.ExecutionContext) error {
	cassandraHelperImpl := ctx.Get(utils.CassandraHelperImpl).(utils.CassandraUtils)
	return cassandraHelperImpl.ExecuteNodetoolCommand(ctx, "nodetool cleanup")
}

func (r *CleanupNodes) Condition(ctx core.ExecutionContext) (bool, error) {
	cassandraHelperImpl := ctx.Get(utils.CassandraHelperImpl).(utils.CassandraUtils)
	return cassandraHelperImpl.IsAllDCsDeployed(ctx), nil
}

type RemoveNodes struct {
	core.DefaultExecutable
}

func (r *RemoveNodes) Execute(ctx core.ExecutionContext) error {
	cassandraHelperImpl := ctx.Get(utils.CassandraHelperImpl).(utils.CassandraUtils)
	spec := ctx.Get(constants.ContextSpec).(*v1alpha1.CassandraDeployment)
	request := ctx.Get(constants.ContextRequest).(reconcile.Request)
	log := ctx.Get(constants.ContextLogger).(*zap.Logger)
	helperImpl := ctx.Get(utils.KubernetesHelperImpl).(core.KubernetesHelper)
	log.Info("Cassandra RemoveNodes step is started")

	dcReplicas := utils.FilterDC(spec.Spec.Cassandra.DeploymentSchema.DataCenters, func(dc *v1alpha1.DataCenter) bool { return dc.Deploy })

	for index, dc := range dcReplicas {
		if dc.Deploy && dc.RemoveNodes != nil && len(dc.RemoveNodes) > 0 {
			for _, nodesToRemove := range dc.RemoveNodes {
				for nodeIndex := range nodesToRemove {
					nodeIndexInt, err := strconv.Atoi(nodeIndex)
					core.PanicError(err, log.Error, fmt.Sprintf("Could not pars nodeIndex from %s", nodeIndex))
					ssIndex := utils.CalcReplicaIndex(dcReplicas, index, nodeIndexInt)
					name := fmt.Sprintf(utils.CassandraReplicaNameFormat, ssIndex)
					err = helperImpl.DeleteStatefulsetAndPods(name, request.Namespace, spec.Spec.WaitTimeout)
					core.PanicError(err, log.Error, "Cassandra statefulset "+name+" deletion failed")

					pods, err := cassandraHelperImpl.GetAllCassandraPods(ctx)
					core.PanicError(err, log.Error, "Cassandra pods listing failed")

					_, err = cassandraHelperImpl.RunSshOnPod(&pods.Items[0], ctx, fmt.Sprintf("nodetool removenode -- %s", nodesToRemove[nodeIndex]))
					if err != nil {
						log.Warn(fmt.Sprintf("Could not perform nodetool removenode. Error: %s"+
							"It might be because the node is already removed. "+
							"Otherwise perform nodetool removenode  -- %s manually", err, nodesToRemove[nodeIndex]))
					}
				}
			}
		}
	}
	return nil
}

func (r *RemoveNodes) Condition(ctx core.ExecutionContext) (bool, error) {
	spec := ctx.Get(constants.ContextSpec).(*v1alpha1.CassandraDeployment)
	nodesToRemoveExist := utils.NewStream(spec.Spec.Cassandra.DeploymentSchema.DataCenters).
		Filter(func(dc interface{}) bool { return dc.(*v1alpha1.DataCenter).Deploy }).
		AnyMatch(func(dc interface{}) bool {
			return dc.(*v1alpha1.DataCenter).RemoveNodes != nil && len(dc.(*v1alpha1.DataCenter).RemoveNodes) > 0
		})

	return core.GetCurrentDeployType(ctx) != core.CleanDeploy && nodesToRemoveExist, nil
}
