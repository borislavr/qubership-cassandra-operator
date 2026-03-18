package utils

import (
	"fmt"

	"github.com/Netcracker/qubership-cql-driver"
	"github.com/gocql/gocql"
	"go.uber.org/zap"
)

type CassandraMetricCollector interface {
	PerformInsertTest() int
	PerformDeleteTest() int
	PerformSelectTest() int
	PrepareSchema() error
}

var _ CassandraMetricCollector = &CassandraMetricCollectorImpl{}

type CassandraMetricCollectorImpl struct {
	Cluster           cql.Cluster
	Logger            *zap.Logger
	TestKeyspace      string
	Replicas          int
	ExternalCassandra bool
	schemaCreated     bool
}

func (m *CassandraMetricCollectorImpl) PerformInsertTest() int {
	queryString := "INSERT INTO %s.test_table (insert_time, key) VALUES (toTimestamp(now()), 'test')"
	insertResult := m.performTest(queryString)
	return insertResult
}

func (m *CassandraMetricCollectorImpl) PerformSelectTest() int {
	queryString := "SELECT insert_time, key FROM %s.test_table WHERE key='test'"
	selectResult := m.performTest(queryString)
	return selectResult
}

func (m *CassandraMetricCollectorImpl) PerformDeleteTest() int {
	queryString := "DELETE FROM %s.test_table WHERE key='test'"
	deleteResult := m.performTest(queryString)
	return deleteResult
}

func (m *CassandraMetricCollectorImpl) PrepareSchema() error {
	session, err := cql.GetSession(m.Cluster, gocql.LocalOne)
	if err != nil {
		return err
	}
	if !m.ExternalCassandra {
		if err := session.Query(fmt.Sprintf("CREATE KEYSPACE if not exists %s  WITH REPLICATION = {'class' : 'NetworkTopologyStrategy', 'replication_factor' : %d }", m.TestKeyspace, m.Replicas)).Exec(false); err != nil {
			return err
		}
	}
	return session.Query(fmt.Sprintf("CREATE TABLE if not exists %s.test_table (key text, insert_time timestamp, PRIMARY KEY((key)))", m.TestKeyspace)).Exec(false)
}

func (m *CassandraMetricCollectorImpl) performTest(query string) int {
	if !m.schemaCreated {
		err := m.PrepareSchema()
		if err != nil {
			m.Logger.Error(fmt.Sprintf("failed to prepare schema for smoke tests: %s", err))
			return 0
		}
		m.schemaCreated = true
	}
	var testResult int = 0
	session, err := cql.GetSession(m.Cluster, gocql.LocalOne)
	if err != nil {
		m.Logger.Error(fmt.Sprintf("failed to create cassandra session: %s", err))
	}

	sessionError := session.Query(fmt.Sprintf(query, m.TestKeyspace)).Exec(false)
	if sessionError != nil {
		testResult = 0
		m.schemaCreated = false // in case if we get error because schema does not exist
		m.Logger.Error(fmt.Sprintf("failed to execute smoke test: %s", sessionError))
	} else {
		testResult = 1
	}
	return testResult
}
