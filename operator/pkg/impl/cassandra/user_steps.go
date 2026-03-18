package cassandra

import (
	"fmt"

	"github.com/Netcracker/qubership-cassandra-operator/api/v1alpha1"
	"github.com/Netcracker/qubership-cassandra-operator/pkg/impl/utils"
	"github.com/Netcracker/qubership-cql-driver"
	"github.com/Netcracker/qubership-nosqldb-operator-core/pkg/constants"
	"github.com/Netcracker/qubership-nosqldb-operator-core/pkg/core"
	"github.com/gocql/gocql"
	"go.uber.org/zap"
)

type CreateSuperUser struct {
	core.DefaultExecutable
	Username string
	Password func() string
}

func (r *CreateSuperUser) Execute(ctx core.ExecutionContext) error {
	cassandraHelperImpl := ctx.Get(utils.CassandraHelperImpl).(utils.CassandraUtils)
	return cassandraHelperImpl.CreateUser(ctx, r.Username, r.Password())
}

func (r *CreateSuperUser) Condition(ctx core.ExecutionContext) (bool, error) {
	log := ctx.Get(constants.ContextLogger).(*zap.Logger)
	cassandraHelperImpl := ctx.Get(utils.CassandraHelperImpl).(utils.CassandraUtils)
	spec := ctx.Get(constants.ContextSpec).(*v1alpha1.CassandraDeployment)
	log.Info("Check if super user already created")

	if pass := ctx.Get(utils.ContextPasswordKey); pass != nil {
		return !cassandraHelperImpl.CheckLogin(ctx, spec.Spec.Cassandra.User, pass.(string)), nil
	}

	return true, nil
}

type DropCassandraDefaultUser struct {
	core.DefaultExecutable
}

func (r *DropCassandraDefaultUser) Execute(ctx core.ExecutionContext) error {
	log := ctx.Get(constants.ContextLogger).(*zap.Logger)
	log.Info("default User deletion")
	cassandraHelperImpl := ctx.Get(utils.CassandraHelperImpl).(utils.CassandraUtils)

	session, sessionErr := cql.GetSession(cassandraHelperImpl.NewClusterBuilder(ctx).Build(), gocql.LocalOne)
	core.PanicError(sessionErr, log.Error, "failed to create cassandra session")
	return session.Query(fmt.Sprintf("DROP ROLE IF EXISTS %s;", utils.Cassandra)).Exec(false)
}

func (r *DropCassandraDefaultUser) Condition(ctx core.ExecutionContext) (bool, error) {
	spec := ctx.Get(constants.ContextSpec).(*v1alpha1.CassandraDeployment)
	return core.GetCurrentDeployType(ctx) == core.CleanDeploy && spec.Spec.Cassandra.User != utils.Cassandra, nil
}
