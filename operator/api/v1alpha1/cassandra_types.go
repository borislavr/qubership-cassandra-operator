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
	"fmt"

	"github.com/Netcracker/qubership-nosqldb-operator-core/pkg/types"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CassandraStatus defines the observed state of Cassandra
type CassandraStatus struct {
	// Important: Run "make" to regenerate code after modifying this file
	Conditions []types.ServiceStatusCondition `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Cassandra is the Schema for the cassandras API
type CassandraDeployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CassandraSpec   `json:"spec,omitempty"`
	Status CassandraStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// CassandraDeploymentList contains a list of Cassandra
type CassandraDeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CassandraDeployment `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CassandraDeployment{}, &CassandraDeploymentList{})
}

type DeploymentSchema struct {
	DataCenters []*DataCenter `json:"dataCenters,omitempty"`
}
type DataCenter struct {
	Deploy              bool                         `json:"deploy,omitempty"`
	Name                string                       `json:"name,omitempty"`
	ClusterDomain       string                       `json:"clusterDomain,omitempty"`
	Replicas            int                          `json:"replicas,omitempty"`
	Seeds               int                          `json:"seeds,omitempty"`
	SeedList            []string                     `json:"seedList,omitempty"`
	RemoveNodes         []map[string]string          `json:"removeNodes,omitempty"`
	Storage             []*types.StorageRequirements `json:"storage,omitempty"`
	Racks               []string                     `json:"racks,omitempty"`
	BroadcastAddress    []string                     `json:"broadcastAddress,omitempty"`
	ListenAddress       []string                     `json:"listenAddress,omitempty"`
	RpcBroadcastAddress []string                     `json:"rpcBroadcastAddress,omitempty"`
	RpcListenAddress    []string                     `json:"rpcListenAddress,omitempty"`
}

func (dc *DataCenter) GetActiveReplicas() []int {
	var activeReplicas []int
	if dc.RemoveNodes != nil {
		for replica := 0; replica < dc.Replicas; replica++ {
			exists := false
			for _, remove := range dc.RemoveNodes {
				for nodeIndex := range remove {
					if fmt.Sprint(replica) == nodeIndex {
						exists = true
					}
				}
			}
			if !exists {
				activeReplicas = append(activeReplicas, replica)
			}
		}

	} else {
		activeReplicas = make([]int, dc.Replicas)
		for replica := 0; replica < dc.Replicas; replica++ {
			activeReplicas[replica] = replica
		}
	}
	return activeReplicas
}

func (dc *DataCenter) GetActiveReplicasLen() int {
	return len(dc.GetActiveReplicas())
}

type CassandraSpec struct {
	DeploymentVersion          string                 `json:"deploymentVersion,omitempty"`
	Recycler                   types.Recycler         `json:"recycler,omitempty"`
	WaitTimeout                int                    `json:"waitTimeout,omitempty"`
	PodSecurityContext         *v1.PodSecurityContext `json:"securityContext,omitempty"`
	Policies                   *Policies              `json:"policies,omitempty" common:"true"`
	Reaper                     Reaper                 `json:"reaper,omitempty" common:"true"`
	TLS                        TLS                    `json:"tls,omitempty" common:"true"`
	Cassandra                  `json:"cassandra"`
	VaultDBEngine              `json:"vaultDBEngine"  common:"true"`
	VaultRegistration          types.VaultRegistration                    `json:"vaultRegistration" common:"true"`
	ServiceAccountName         string                                     `json:"serviceAccountName"`
	IpV6                       bool                                       `json:"ipV6,omitempty"`
	StopOnFailedResourceUpdate bool                                       `json:"stopOnFailedResourceUpdate,omitempty"`
	ConsulRegistration         types.ConsulRegistration                   `json:"consulRegistration,omitempty"`
	ConsulDiscoverySettings    map[string]*types.AgentServiceRegistration `json:"consulDiscoverySettings,omitempty"`
	GocqlConnectTimeout        int                                        `json:"gocqlConnectTimeout"`
	GocqlTimeout               int                                        `json:"gocqlTimeout"`
	ImagePullPolicy            v1.PullPolicy                              `json:"imagePullPolicy,omitempty" common:"true"`
	DeletePVConUninstall       bool                                       `json:"deletePVConUninstall,omitempty"`
	ArtifactDescriptorVersion  string                                     `json:"artifactDescriptorVersion,omitempty"`
	PartOf                     string                                     `json:"partOf,omitempty"`
	ManagedBy                  string                                     `json:"managedBy,omitempty"`
	Instance                   string                                     `json:"instance,omitempty"`
}

type Cassandra struct {
	Install           bool                     `json:"install,omitempty"`
	User              string                   `json:"username,omitempty"`
	Password          string                   `json:"password,omitempty"`
	SecretName        string                   `json:"secretName,omitempty"`
	Resources         *v1.ResourceRequirements `json:"resources,omitempty"`
	DockerImage       string                   `json:"dockerImage,omitempty"`
	DeploymentSchema  *DeploymentSchema        `json:"deploymentSchema,omitempty"`
	Configuration     string                   `json:"configuration,omitempty"`
	HostNetwork       bool                     `json:"hostNetwork,omitempty"`
	PriorityClassName string                   `json:"priorityClassName,omitempty"`
	Envs              map[string]string        `json:"envs,omitempty"`
	AuditLogEnabled   bool                     `json:"auditLogEnabled,omitempty"`
	SmoketestKeyspace string                   `json:"smoketestKeyspace,omitempty"`
	// enables audit log in cassandra 3
	CommitlogArchiving CommitlogArchiving `json:"commitlogArchiving,omitempty"`
	Affinity           *v1.Affinity       `json:"affinity,omitempty"`
}

type CommitlogArchiving struct {
	Enabled bool `json:"enabled,omitempty"`
}

type Reaper struct {
	Install              bool                     `json:"install,omitempty"`
	DockerImage          string                   `json:"dockerImage,omitempty"`
	IngressHost          string                   `json:"ingressHost,omitempty"`
	Port                 int                      `json:"port,omitempty"`
	Type                 string                   `json:"type,omitempty"`
	Username             string                   `json:"username,omitempty"`
	SecretName           string                   `json:"secretName,omitempty"`
	Envs                 map[string]string        `json:"envs,omitempty"`
	Resources            *v1.ResourceRequirements `json:"resources,omitempty"`
	TruststoreSecretName string                   `json:"truststoreSecretName,omitempty"`
}

type Policies struct {
	Tolerations []v1.Toleration `json:"tolerations,omitempty"`
}

type VaultDBEngine struct {
	Enabled         bool   `json:"enabled"`
	Name            string `json:"name,omitempty"`
	Role            string `json:"role,omitempty"`
	PluginName      string `json:"pluginName,omitempty"`
	ProtocolVersion int    `json:"protocolVersion,omitempty"`
	AllowedRoles    string `json:"allowedRoles,omitempty"`
	ConnectTimeout  string `json:"connectTimeout,omitempty"`
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
