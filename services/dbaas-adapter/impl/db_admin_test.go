package impl

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/Netcracker/qubership-cassandra-dbaas-adapter/go/impl/cassandra"
	"github.com/Netcracker/qubership-cassandra-dbaas-adapter/go/impl/cassandra/mocks"
	mUtils "github.com/Netcracker/qubership-cassandra-dbaas-adapter/go/utils"
	"github.com/Netcracker/qubership-dbaas-adapter-core/pkg/dao"
	"github.com/gocql/gocql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestCassandraDbAdministration(t *testing.T) {
	ctx := context.Background()
	cluster := mocks.NewClusterInterface(t)
	configuration := mocks.NewCassandraConfiguration(t)
	casService := mocks.NewCassandraService(t)
	session := mocks.NewSession(t)
	query := mocks.NewQueryInterface(t)

	cluster.On("NewSession").Return(session, nil)

	session.On("Query", mock.Anything, mock.Anything, mock.Anything).Return(query)
	session.On("Close")

	query.On("Exec", mock.Anything).Return(nil)

	configuration.On("GetCluster", mock.Anything, mock.Anything).Return(cluster, nil)

	logger := GetLogger(true)
	sessionService := &cassandra.SessionServiceImpl{
		Logger:             logger,
		DefaultKeyspace:    "system",
		DefaultConsistency: gocql.LocalOne,
		Configuration:      configuration,
	}

	dbAdmin := &CassandraDbAdministration{
		logger:           logger,
		supportedRoles:   []string{"admin"},
		apiVersion:       "v2",
		sessionService:   sessionService,
		cassandraService: casService,
	}

	dbname := "foobar"
	replicationValue := "test"
	prefix := ""
	dbRequest := dao.DbCreateRequest{
		DbName:     dbname,
		NamePrefix: &prefix,
		Settings:   map[string]interface{}{"replication": replicationValue},
		Metadata:   map[string]interface{}{"classifier": map[string]interface{}{"microserviceName": "test-ms123", "namespace": "test-ns123", "scope": "service"}, "microserviceName": "test-ms123"},
	}

	t.Run("GrantPermissions failed, rollback expected", func(t *testing.T) {
		casService.On("CreateKeyspace", mock.Anything, dbname, replicationValue).Return(nil)
		casService.On("CreateTable", mock.Anything, dbname, mock.Anything).Return(nil)

		casService.On("CreateRole", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		casService.On("GrantPermissions", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("Failed to grant permissions"))

		casService.On("DropResource", mock.Anything, "keyspace", dbname).Return(nil)
		casService.On("DropResource", mock.Anything, "role", mock.Anything).Return(nil)

		assert.Panics(t, func() {
			dbAdmin.CreateDatabase(ctx, dbRequest)
		})
	})

}

func GetLogger(debug bool) *zap.Logger {
	var atom zap.AtomicLevel
	if debug {
		atom = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	} else {
		atom = zap.NewAtomicLevel()
	}
	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "timestamp"
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder

	logger := zap.New(zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderCfg),
		zapcore.Lock(os.Stdout),
		atom,
	))
	defer logger.Sync()
	return logger
}

func TestValidateRequestParamsAndGetLogicalDbName_NoPrefix(t *testing.T) {
	requestOnCreateDb := dao.DbCreateRequest{
		Metadata: map[string]interface{}{"classifier": map[string]interface{}{"microserviceName": "test-ms123", "namespace": "test-ms123", "scope": "service"},
			"microserviceName": "test-ms123"},
	}

	adm := &CassandraDbAdministration{}

	name, err := adm.validateRequestParamsAndGetLogicalDbName(requestOnCreateDb)
	assert.NoError(t, err)
	assert.Contains(t, name, "test_ms123_test_ms123")
}

func TestValidateRequestParamsAndGetLogicalDbName_YesPrefix(t *testing.T) {
	prefix := "testPrefix"
	requestOnCreateDb := dao.DbCreateRequest{
		NamePrefix: &prefix,
		Metadata: map[string]interface{}{"classifier": map[string]interface{}{"microserviceName": "test-ms123", "namespace": "test-ms123", "scope": "service"},
			"microserviceName": "test-ms123"},
	}

	adm := &CassandraDbAdministration{}

	name, err := adm.validateRequestParamsAndGetLogicalDbName(requestOnCreateDb)
	assert.NoError(t, err)
	assert.Contains(t, name, "testPrefix")
	assert.NotContains(t, name, "test-ms123_test-ms123")
}
func TestCassandraDbAdministration_GetSupportedRoles(t *testing.T) {
	c := &CassandraDbAdministration{
		supportedFeatures: map[string]bool{mUtils.FeatureMultiUsers: false},
	}

	roles := c.GetSupportedRoles()

	assert.Equal(t, []string{"admin"}, roles)
}
