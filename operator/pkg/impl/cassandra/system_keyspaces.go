package cassandra

import (
	"fmt"

	"github.com/Netcracker/qubership-cassandra-operator/pkg/impl/utils"
	"github.com/Netcracker/qubership-cql-driver"
	"github.com/Netcracker/qubership-nosqldb-operator-core/pkg/constants"
	"github.com/Netcracker/qubership-nosqldb-operator-core/pkg/core"
	"github.com/gocql/gocql"
	"go.uber.org/zap"
)

type UpdateSystemKeyspacesTopology struct {
	core.DefaultExecutable
}

func (r *UpdateSystemKeyspacesTopology) Execute(ctx core.ExecutionContext) error {
	log := ctx.Get(constants.ContextLogger).(*zap.Logger)
	cassandraHelperImpl := ctx.Get(utils.CassandraHelperImpl).(utils.CassandraUtils)

	replication := cassandraHelperImpl.GetReplicationFactor(ctx)
	log.Info("system_auth replication will be updated with: " + replication)

	cluster := cassandraHelperImpl.NewClusterBuilder(ctx).Build()
	session, sessionErr := cql.GetSession(cluster, gocql.Quorum)
	core.PanicError(sessionErr, log.Error, "failed to create cassandra session")

	// if sessionError != nil {
	// 	log.Warn("Falling back to default credentials")
	// 	cluster = cassandraHelperImpl.NewClusterBuilder(ctx).WithUser(utils.Cassandra).
	// 		WithPassword(func() string { return utils.Cassandra }).WithConsistency(gocql.LocalOne).Build()
	// }

	utils.NewStream(utils.SystemKeyspaces).ForEach(func(keyspace interface{}) {
		session.Query(fmt.Sprintf(
			"alter KEYSPACE %s WITH replication = {'class': 'NetworkTopologyStrategy', %s}", keyspace, replication)).Exec(true)
	})

	list, err := cassandraHelperImpl.GetAllCassandraPods(ctx)
	core.PanicError(err, log.Error, "Cassandra pods listing failed")

	if len(list.Items) == 0 {
		return &core.ExecutionError{Msg: "Cassandra pods not found"}
	}

	for _, cassandraPod := range list.Items {
		for _, keyspace := range utils.SystemKeyspaces {
			command := fmt.Sprintf("nodetool repair -full %s 2> /dev/null", keyspace)
			_, err := cassandraHelperImpl.RunSshOnPod(&cassandraPod, ctx, command)
			if err != nil {
				log.Warn(fmt.Sprintf("Could not perform nodetool repair. Error: %s"+
					"If needed perform %s manually", err, command))
			}
		}
	}

	return nil

}

func (r *UpdateSystemKeyspacesTopology) Condition(ctx core.ExecutionContext) (bool, error) {
	return core.GetCurrentDeployType(ctx) == core.CleanDeploy, nil
}
