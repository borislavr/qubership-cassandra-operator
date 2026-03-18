/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"github.com/Netcracker/qubership-nosqldb-operator-core/pkg/types"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Cassandra struct {
	Install          bool              `json:"install,omitempty"`
	User             string            `json:"username,omitempty"`
	Password         string            `json:"password,omitempty"`
	SecretName       string            `json:"secretName,omitempty"`
	Host             string            `json:"host,omitempty"`
	DefaultKeyspace  string            `json:"defaultKeyspace,omitempty"`
	Consistency      string            `json:"consistency,omitempty"`
	TLS              bool              `json:"tls,omitempty"`
	Port             int               `json:"port,omitempty"`
	DeploymentSchema *DeploymentSchema `json:"deploymentSchema,omitempty"`
}

type DeploymentSchema struct {
	DataCenters []*DataCenter `json:"dataCenters,omitempty"`
}

type DataCenter struct {
	Name     string `json:"name,omitempty"`
	Replicas int    `json:"replicas,omitempty"`
	Deploy   bool   `json:"deploy,omitempty"`
}

type Policies struct {
	Tolerations []v1.Toleration `json:"tolerations,omitempty"`
}

type CassandraServiceSpec struct {
	Cassandra                  Cassandra              `json:"cassandra" common:"true"`
	DeploymentVersion          string                 `json:"deploymentVersion,omitempty"`
	Recycler                   types.Recycler         `json:"recycler,omitempty"`
	WaitTimeout                int                    `json:"waitTimeout,omitempty"`
	PodSecurityContext         *v1.PodSecurityContext `json:"securityContext,omitempty"`
	Policies                   *Policies              `json:"policies,omitempty" common:"true"`
	TLS                        TLS                    `json:"tls,omitempty" common:"true"`
	Backup                     `json:"backupDaemon"`
	Dbaas                      `json:"dbaas"`
	Monitoring                 `json:"monitoringAgent"`
	RobotTests                 `json:"robotTests"`
	VaultRegistration          types.VaultRegistration                    `json:"vaultRegistration" common:"true"`
	ServiceAccountName         string                                     `json:"serviceAccountName"`
	IpV6                       bool                                       `json:"ipV6,omitempty"`
	StopOnFailedResourceUpdate bool                                       `json:"stopOnFailedResourceUpdate,omitempty"`
	ConsulRegistration         types.ConsulRegistration                   `json:"consulRegistration,omitempty"`
	ConsulDiscoverySettings    map[string]*types.AgentServiceRegistration `json:"consulDiscoverySettings,omitempty"`
	GocqlConnectTimeout        int                                        `json:"gocqlConnectTimeout"`
	GocqlTimeout               int                                        `json:"gocqlTimeout"`
	ImagePullPolicy            v1.PullPolicy                              `json:"imagePullPolicy,omitempty" common:"true"`
	AWSKeyspaces               AWSKeyspaces                               `json:"awsKeyspaces,omitempty" common:"true"`
	ArtifactDescriptorVersion  string                                     `json:"artifactDescriptorVersion,omitempty"`
	PartOf                     string                                     `json:"partOf,omitempty"`
	ManagedBy                  string                                     `json:"managedBy,omitempty"`
	Instance                   string                                     `json:"instance,omitempty"`
	DeletePVConUninstall       bool                                       `json:"deletePVConUninstall,omitempty"`
}

// CassandraServiceStatus defines the observed state of CassandraService
// +k8s:openapi-gen=true
type CassandraServiceStatus struct {
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	Conditions []types.ServiceStatusCondition `json:"conditions,omitempty"`
}

type TLS struct {
	//if TLS encryption should be enabled
	Enabled bool `json:"enabled,omitempty"`
	//if TLS encryption is optional. If `true` then Cassandra cluster will accept TLS and non TLS connections.
	Optional bool `json:"optional,omitempty"`
	//a name of Kubernetes secret that holds a CA certificate, a Signed Cassandra sertificate and a private key.
	RootCASecretName string `json:"rootCASecretName,omitempty"`
	//a key in the Kubernetes secret `tls.rootCASecretName` that holds the CA certificate.
	RootCAFileName string `json:"rootCAFileName,omitempty"`
	//a key in the Kubernetes secret `tls.rootCASecretName` that holds the Signed Cassandra sertificate.
	SignedCRTFileName string `json:"signedCRTFileName,omitempty"`
	//a key in the Kubernetes secret `tls.rootCASecretName` that holds the private key.
	PrivateKeyFileName string `json:"privateKeyFileName,omitempty"`
	//a password to Cassandra keystore.
	KeystorePass string `json:"keystorePass,omitempty"`
}

type BackupDaemonTLS struct {
	// a name of Kubernetes secret that holds a CA certificate, a Signed Cassandra Backup Daemon sertificate and a private key.
	BackupDaemonCASecretName string `json:"backupDaemonCASecretName,omitempty"`
}

type DbaasAdapterTLS struct {
	// a name of Kubernetes secret that holds a CA certificate, a Signed Cassandra Dbaas Adapter sertificate and a private key.
	DbaasAdapterCASecretName string `json:"dbaasAdapterCASecretName,omitempty"`
}

type Backup struct {
	Install          bool              `json:"install,omitempty"`
	LegacyMode       bool              `json:"legacyMode,omitempty"`
	StorageDirectory string            `json:"storageDirectory,omitempty"`
	DockerImage      string            `json:"dockerImage,omitempty"`
	NodeLabels       map[string]string `json:"nodeLabels,omitempty"`
	BackupSchedule   string            `json:"backupSchedule,omitempty"`
	// Schedule for periodic granular backups
	GranularBackupSchedule string `json:"granularBackupSchedule,omitempty"`
	// List of dbs for scheduled granular backups
	GranularBackupScheduledDbs []string                   `json:"granularBackupScheduledDbs,omitempty"`
	EvictionPolicy             string                     `json:"evictionPolicy,omitempty"`
	GranularEvictionPolicy     string                     `json:"granularEvictionPolicy,omitempty"`
	User                       string                     `json:"username,omitempty"`
	SecretName                 string                     `json:"secretName,omitempty"`
	Storage                    *types.StorageRequirements `json:"storage,omitempty"`
	Resources                  *v1.ResourceRequirements   `json:"resources,omitempty"`
	PriorityClassName          string                     `json:"priorityClassName,omitempty"`
	S3                         S3backup                   `json:"s3,omitempty"`
	TLS                        BackupDaemonTLS            `json:"tls,omitempty"`
}

type S3backup struct {
	Enabled         bool   `json:"enabled,omitempty"`
	SecretName      string `json:"secretName,omitempty"`
	BucketName      string `json:"bucketName,omitempty"`
	AccessKeyId     string `json:"accessKeyId,omitempty"`
	AccessKeySecret string `json:"accessKeySecret,omitempty"`
	EndpointUrl     string `json:"endpointUrl,omitempty"`
	SslVerify       bool   `json:"sslVerify,omitempty"`
	SslSecretName   string `json:"sslSecretName,omitempty"`
	SslCert         string `json:"sslCert,omitempty"`
}

type DbaasAdapterCredentials struct {
	Username   string `json:"username,omitempty"`
	SecretName string `json:"secretName,omitempty"`
}

type DbaasAggregatorCredentials struct {
	Username                           string `json:"username,omitempty"`
	PhysicalDatabaseIdentifier         string `json:"physicalDatabaseIdentifier,omitempty"`
	DbaasAggregatorRegistrationAddress string `json:"dbaasAggregatorRegistrationAddress,omitempty"`
	SecretName                         string `json:"secretName,omitempty"`
}

type Dbaas struct {
	Install           bool                        `json:"install,omitempty"`
	DockerImage       string                      `json:"dockerImage,omitempty"`
	NodeLabels        map[string]string           `json:"nodeLabels,omitempty"`
	Resources         *v1.ResourceRequirements    `json:"resources,omitempty"`
	Adapter           *DbaasAdapterCredentials    `json:"adapter,omitempty"`
	Aggregator        *DbaasAggregatorCredentials `json:"aggregator,omitempty"`
	PriorityClassName string                      `json:"priorityClassName,omitempty"`
	ApiVersion        string                      `json:"apiVersion,omitempty"`
	MultiUsers        bool                        `json:"multiUsers,omitempty"`
	//default topology strategy for keyspaces created via Dbaas. The default values is `"{'class':'SimpleStrategy','replication_factor': 1 }"`. This parameter is ignored if `dbaas.allDCTopologyStrategy` is `true`. The value can be overridden in a request body to Dbaas.
	TopologyStrategy string          `json:"topologyStrategy,omitempty"`
	TLS              DbaasAdapterTLS `json:"tls,omitempty"`
}

type AWSKeyspaces struct {
	Install    bool   `json:"install,omitempty"`
	SecretName string `json:"secretName,omitempty"`
	Host       string `json:"host,omitempty"`
}

type RobotTests struct {
	Install           bool                     `json:"install,omitempty"`
	DockerImage       string                   `json:"dockerImage,omitempty"`
	Resources         *v1.ResourceRequirements `json:"resources,omitempty"`
	Tags              string                   `json:"tags,omitempty"`
	Iteration         string                   `json:"iteration,omitempty"`
	PrometheusUrl     string                   `json:"prometheusUrl,omitempty"`
	Args              []string                 `json:"args,omitempty"`
	ReplicationFactor int                      `json:"replicationFactor,omitempty"`
	AttemptsNumber    int                      `json:"attemptsNumber,omitempty"`
	NodeLabels        map[string]string        `json:"nodeLabels,omitempty"`
}

type Monitoring struct {
	Install            bool   `json:"install,omitempty"`
	MonitoringInterval string `json:"monitoringInterval,omitempty"`
	CollectionJitter   string `json:"collectionJitter,omitempty"`
	FlushInterval      string `json:"flushInterval,omitempty"`
	FlushJitter        string `json:"flushJitter,omitempty"`
	MetricCollector    string `json:"metricCollector,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion

// CassandraSupplService is the Schema for the cassandrasupplservice API
type CassandraSupplService struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CassandraServiceSpec   `json:"spec,omitempty"`
	Status CassandraServiceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// CassandraSupplServiceList contains a list of CassandraService
type CassandraSupplServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CassandraSupplService `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CassandraSupplService{}, &CassandraSupplServiceList{})
}
