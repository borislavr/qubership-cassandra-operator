package fiber

import (
	"fmt"

	"github.com/Netcracker/qubership-cassandra-operator/api/v1alpha1"
	"github.com/Netcracker/qubership-cassandra-operator/pkg/impl/utils"
	"github.com/Netcracker/qubership-nosqldb-operator-core/pkg/constants"
	"github.com/Netcracker/qubership-nosqldb-operator-core/pkg/core"
	"github.com/gocql/gocql"
	"go.uber.org/zap"
)

type CassandraFiberService interface {
	FlushInMemoryData() error
	PerformInsertTest() int
	PerformDeleteTest() int
	PerformSelectTest() int
}

type CassandraFiberServiceImpl struct {
	ctx          core.ExecutionContext
	metricHelper utils.CassandraMetricCollector
}

func NewCassandraFiberService(ctx core.ExecutionContext) CassandraFiberService {
	spec := ctx.Get(constants.ContextSpec).(*v1alpha1.CassandraDeployment)
	cassandraHelperImpl := ctx.Get(utils.CassandraHelperImpl).(utils.CassandraUtils)
	logger := ctx.Get(constants.ContextLogger).(*zap.Logger)
	replicas := 1
	keyspaceName := spec.Spec.Cassandra.SmoketestKeyspace
	if spec.Spec.Cassandra.Install {
		currentDc := utils.NewStream(spec.Spec.Cassandra.DeploymentSchema.DataCenters).FindFirst(func(dc interface{}) bool {
			return dc.(*v1alpha1.DataCenter).Deploy
		}).(*v1alpha1.DataCenter)
		replicas = utils.Min(currentDc.Replicas, 3)
		keyspaceName = fmt.Sprintf("%s_%s", keyspaceName, currentDc.Name)
	}

	cassandraMetricHelperImpl := &utils.CassandraMetricCollectorImpl{
		Cluster: cassandraHelperImpl.NewClusterBuilder(ctx).
			WithTLSEnabled(spec.Spec.TLS.Enabled).WithConsistency(gocql.LocalOne).Build(), //TODO get rid of gocql
		TestKeyspace: keyspaceName,
		Replicas:     replicas,
		Logger:       logger,
	}
	return &CassandraFiberServiceImpl{ctx: ctx, metricHelper: cassandraMetricHelperImpl}
}

func (m *CassandraFiberServiceImpl) FlushInMemoryData() error {
	log := m.ctx.Get(constants.ContextLogger).(*zap.Logger)
	cassandraHelperImpl := m.ctx.Get(utils.CassandraHelperImpl).(utils.CassandraUtils)

	err := cassandraHelperImpl.ExecuteFlushData(m.ctx)
	if err != nil {
		core.PanicError(err, log.Error, "Cassandra pods flush command failed")
		return err
	}

	return nil
}

func (m *CassandraFiberServiceImpl) PerformInsertTest() int {
	return m.metricHelper.PerformInsertTest()
}

func (m *CassandraFiberServiceImpl) PerformSelectTest() int {
	return m.metricHelper.PerformSelectTest()
}

func (m *CassandraFiberServiceImpl) PerformDeleteTest() int {
	return m.metricHelper.PerformDeleteTest()
}

func (m *CassandraFiberServiceImpl) PrepareSchema() error {
	return m.metricHelper.PrepareSchema()
}
