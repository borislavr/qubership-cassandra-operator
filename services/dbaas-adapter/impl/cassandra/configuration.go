package cassandra

import (
	"crypto/tls"
	"time"

	mUtils "github.com/Netcracker/qubership-cassandra-dbaas-adapter/go/utils"
	"github.com/Netcracker/qubership-dbaas-adapter-core/pkg/utils"
	"github.com/gocql/gocql"
)

type CassandraConfiguration interface {
	GetHost() string
	GetPort() int
	GetCluster(keyspace string, consistency gocql.Consistency) ClusterInterface
	GetUser() string
	GetDefaultTopology() string
	IsTLSEnabled() bool
}

type CassandraConfigurationImpl struct {
	Hostname       string
	Port           int
	User           string
	Pass           string
	ConnectTimeout int
	Timeout        int
	Topology       string
}

var _ CassandraConfiguration = &CassandraConfigurationImpl{}

func (r *CassandraConfigurationImpl) GetUser() string {
	return r.User
}

func (r *CassandraConfigurationImpl) GetHost() string {
	return r.Hostname
}

func (r *CassandraConfigurationImpl) GetDefaultTopology() string {
	return r.Topology
}

func (r *CassandraConfigurationImpl) GetPort() int {
	return r.Port
}

func (r *CassandraConfigurationImpl) GetCluster(keyspace string, consistency gocql.Consistency) ClusterInterface {
	cluster := gocql.NewCluster(r.Hostname)
	cluster.Port = r.Port
	cluster.Authenticator = gocql.PasswordAuthenticator{
		Username: r.User,
		Password: r.Pass,
	}
	cluster.ProtoVersion = 4
	cluster.Keyspace = keyspace
	cluster.Consistency = consistency
	cluster.Timeout = time.Duration(r.Timeout) * time.Second
	cluster.ConnectTimeout = time.Duration(r.ConnectTimeout) * time.Second

	if utils.IsTLSEnabledForMainService() {
		if cert := mUtils.GetCACert(); cert != "" {
			cluster.SslOpts = &gocql.SslOptions{
				CaPath: cert,
			}
		} else {
			cluster.SslOpts = &gocql.SslOptions{Config: &tls.Config{MinVersion: tls.VersionTLS12, InsecureSkipVerify: true}}
		}

	}
	return &Cluster{cluster}
}

func (r *CassandraConfigurationImpl) IsTLSEnabled() bool {
	return utils.IsTLSEnabledForMainService()
}
