package cassandra

import (
	"fmt"
	"sync"

	"github.com/Netcracker/qubership-dbaas-adapter-core/pkg/utils"
	"github.com/gocql/gocql"
	"go.uber.org/zap"
)

type SessionResult func(sessionInterface Session) interface{}

type SessionService interface {
	GetConfiguration() CassandraConfiguration
	NewSession() Session
	NewAutoCloseSession(sessionFunc SessionResult) interface{}
	UseActiveSession(sessionFunc SessionResult) interface{} //todo return error
	NewSessionToKeyspace(keyspace string, consistency *gocql.Consistency) Session
	NewAutoCloseSessionToKeyspace(sessionFunc SessionResult, keyspace string, consistency *gocql.Consistency) interface{}
}

type SessionServiceImpl struct {
	Logger             *zap.Logger
	Configuration      CassandraConfiguration
	DefaultKeyspace    string
	DefaultConsistency gocql.Consistency
	activeSession      Session
	activeSessionMutex sync.Mutex
}

func (r *SessionServiceImpl) GetConfiguration() CassandraConfiguration {
	return r.Configuration
}

func (r *SessionServiceImpl) NewSessionToKeyspace(
	keyspace string,
	consistency *gocql.Consistency) Session {
	con := r.DefaultConsistency
	if consistency != nil {
		con = *consistency
	}
	k := r.DefaultKeyspace
	if keyspace != "" {
		k = keyspace
	}
	r.Logger.Debug(fmt.Sprintf("Requested new session to %s keyspace with %s consistency", k, con))
	cluster := r.Configuration.GetCluster(k, con)
	session, err := cluster.NewSession()
	utils.PanicError(err, r.Logger.Error, "Could not create cassandra session")
	return session
}

func (r *SessionServiceImpl) NewAutoCloseSessionToKeyspace(
	sessionFunc SessionResult,
	keyspace string,
	consistency *gocql.Consistency) interface{} {
	session := r.NewSessionToKeyspace(keyspace, consistency)
	defer session.Close()
	return sessionFunc(session)
}

func (r *SessionServiceImpl) NewSession() Session {
	return r.NewSessionToKeyspace("", nil)
}

func (r *SessionServiceImpl) NewAutoCloseSession(sessionFunc SessionResult) interface{} {
	return r.NewAutoCloseSessionToKeyspace(sessionFunc, "", nil)
}

func (r *SessionServiceImpl) UseActiveSession(sessionFunc SessionResult) interface{} {
	r.activeSessionMutex.Lock()
	defer r.activeSessionMutex.Unlock()
	if r.activeSession == nil {
		r.Logger.Debug("Active Cassandra session not found. Will be created new one")
		r.activeSession = r.NewSessionToKeyspace("", nil)
	}
	return sessionFunc(r.activeSession)
}
