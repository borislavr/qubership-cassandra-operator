package tests

import (
	"context"
	"fmt"
	"testing"

	"github.com/Netcracker/qubership-cassandra-operator/api/v1alpha1"
	"github.com/Netcracker/qubership-cassandra-operator/pkg/impl"
	"github.com/Netcracker/qubership-cassandra-operator/pkg/impl/cassandra"
	"github.com/Netcracker/qubership-cassandra-operator/pkg/impl/utils"
	"github.com/Netcracker/qubership-cql-driver"
	"github.com/Netcracker/qubership-cql-driver/mocks"
	"github.com/Netcracker/qubership-nosqldb-operator-core/pkg/constants"
	"github.com/Netcracker/qubership-nosqldb-operator-core/pkg/core"
	mTypes "github.com/Netcracker/qubership-nosqldb-operator-core/pkg/types"
	mVault "github.com/Netcracker/qubership-nosqldb-operator-core/pkg/vault"
	"go.uber.org/zap"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	v1app "k8s.io/api/apps/v1"
	v1core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type TestUtilsImpl struct {
	core.DefaultKubernetesHelperImpl
}

func (r *TestUtilsImpl) WaitForPVCBound(pvcName string, namespace string, waitSeconds int) error {
	return nil
}

func (r *TestUtilsImpl) WaitForDeploymentReady(deployName string, namespace string, waitSeconds int) error {
	return nil
}

func (r *TestUtilsImpl) WaitForTestsReady(deployName string, namespace string, waitSeconds int) error {
	return nil
}

func (r *TestUtilsImpl) WaitForPodsReady(labelSelectors map[string]string, namespace string, numberOfPods int, waitSeconds int) error {
	return nil
}

func (r *TestUtilsImpl) WaitForPodsCompleted(labelSelectors map[string]string, namespace string, numberOfPods int, waitSeconds int) error {
	return nil
}

func (r *TestUtilsImpl) WaitPodsCountByLabel(labelSelectors map[string]string, namespace string, numberOfPods int, waitSeconds int) error {
	return nil
}

func (r *TestUtilsImpl) ExecRemote(log *zap.Logger, kubeConfig *rest.Config, podName string, namespace string, containerName string, command string, args []string) (string, error) {
	return "", nil
}

func (r *TestUtilsImpl) GetPodLogs(kubeConfig *rest.Config, podName string, namespace string, containerName string, tailLines *int64, previous bool) (string, error) {

	return "", nil
}

func (r *TestUtilsImpl) ListPods(namespace string, labelSelectors map[string]string) (*v1core.PodList, error) {
	return &v1core.PodList{
		Items: []v1core.Pod{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod",
					Namespace: namespace,
					Labels:    labelSelectors,
				},
				Spec: v1core.PodSpec{
					Containers: []v1core.Container{
						{
							Name: "pod",
						},
					},
				},
			},
		},
	}, nil
}

type TestCassandraUtilsImpl struct {
	utils.CassandraUtilsImpl
	t *testing.T
}

func (r *TestCassandraUtilsImpl) NewClusterBuilder(ctx core.ExecutionContext) cql.ClusterBuilder {
	var cluster cql.Cluster = &mocks.TestCluster{}
	clusterBuiler := mocks.ClusterBuilder{}
	clusterBuiler.On("Build").Return(cluster)
	return &clusterBuiler
}

func (r *TestCassandraUtilsImpl) CheckLogin(ctx core.ExecutionContext, username string, password string) bool {
	return true
}

func (r *TestCassandraUtilsImpl) CreateUser(ctx core.ExecutionContext, username string, password string) error {
	return nil
}
func (r *TestCassandraUtilsImpl) UpdateUserPass(ctx core.ExecutionContext, username, oldPassword, newPassword string) error {
	return nil
}

type CaseStruct struct {
	name                          string
	nameSpace                     string
	executor                      core.Executor
	builder                       core.ExecutableBuilder
	ctx                           core.ExecutionContext
	ctxToReplaceAfterServiceBuilt map[string]interface{}
	RunTestFunc                   func() error
	ReadResultFunc                func(t *testing.T, err error)
	ReadErrorFunc                 func(t *testing.T, err error) error
}

func GenerateDefaultCassandraTestCase(
	t *testing.T,
	vaultImpl mVault.VaultHelper,
	testName string,
	CassandraDeploymentSpec *v1alpha1.CassandraDeployment,
	runtimeBuilder RuntimeObjectBuilder,
	nameSpace string,
	nameSpaceRequestName string,
) CaseStruct {
	client := fake.NewFakeClient(runtimeBuilder.runtimeObjects...)
	utilsHelp := &TestUtilsImpl{}
	utilsHelp.ForceKey = true
	// Because there is empty runtime Scheme
	utilsHelp.OwnerKey = false
	utilsHelp.Client = client

	cassandraHelper := &TestCassandraUtilsImpl{}
	cassandraHelper.KubernetesHelperImpl = utilsHelp

	caseStruct := CaseStruct{
		name:      testName,
		nameSpace: nameSpace,
		executor:  core.DefaultExecutor(),
		builder:   &impl.CassandraBuilder{},
		ctx: core.GetExecutionContext(map[string]interface{}{
			constants.ContextSpec:   CassandraDeploymentSpec,
			constants.ContextSchema: &runtime.Scheme{},
			constants.ContextRequest: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: nameSpace,
					Name:      nameSpaceRequestName,
				},
			},
			constants.ContextClient:                client,
			constants.ContextKubeClient:            &rest.Config{},
			constants.ContextLogger:                core.GetLogger(true),
			"contextResourceOwner":                 CassandraDeploymentSpec, //todo hardcode replace
			constants.ContextServiceDeploymentInfo: map[string]string{},
			constants.ContextVault:                 vaultImpl,
			constants.ContextHashConfigMap:         "random",
		}),
		ctxToReplaceAfterServiceBuilt: map[string]interface{}{
			utils.KubernetesHelperImpl: utilsHelp,
			utils.CassandraHelperImpl:  cassandraHelper,
		},
		ReadResultFunc: func(t *testing.T, err error) {
			if err != nil {
				// Some error happened
				t.Error(err)
			}
		},
	}

	return caseStruct
}

func GenerateDefaultCassandraWrapper(
	t *testing.T,
	vaultImpl mVault.VaultHelper,
	testName string,
	replicas int,
	dcCount int,
) CaseStruct {

	nameSpace := "cassandra-namespace"
	nameSpaceRequestName := "cassandra-name"

	runtimeBuilder := RuntimeObjectBuilder{namespace: nameSpace}

	runtimeBuilder.GenerateSecrets("cassandra-secret", "admin", "admin")

	dcs := []*v1alpha1.DataCenter{}

	for i := 0; i < dcCount; i++ {
		cassandraDCPvNames, cassandraDCNodes := runtimeBuilder.GeneratePVs(utils.CassandraDCPvcNameFormat+"-%v", "node-%v", i, replicas)
		cassandraAdditionalDCPvNames, cassandraAdditionalDCNodes := runtimeBuilder.GeneratePVs(utils.CassandraDCPvcNameFormat+"-%v-1", "node-%v", i, replicas)

		dcs = append(dcs,
			&v1alpha1.DataCenter{
				Name:     fmt.Sprintf("dcName%v", i),
				Replicas: replicas,
				Seeds:    1,
				Deploy:   true,
				Storage: []*mTypes.StorageRequirements{
					{
						Size:       []string{"5Gi"},
						Volumes:    cassandraDCPvNames,
						NodeLabels: cassandraDCNodes,
						MountSettings: &v1core.VolumeMount{
							Name:      "data",
							MountPath: "/var/lib/cassandra/data",
						},
					},
					{
						Size:       []string{"4Gi"},
						Volumes:    cassandraAdditionalDCPvNames,
						NodeLabels: cassandraAdditionalDCNodes,
						MountSettings: &v1core.VolumeMount{
							Name:      "commit-log",
							MountPath: "/var/lib/anotherpath/commitlog",
						},
					},
				},
			},
		)
	}

	configmapData := map[string][]byte{
		utils.Config: []byte(`cluster_name: 'cassandra_cluster'
num_tokens: 256
hinted_handoff_enabled: true
data_file_directories:
- /var/lib/cassandra/data
hinted_handoff_throttle_in_kb: 1024
max_hints_delivery_threads: 2
hints_directory: /var/lib/cassandra/hints
hints_flush_period_in_ms: 10000
max_hints_file_size_in_mb: 128
batchlog_replay_throttle_in_kb: 1024
authenticator: PasswordAuthenticator
authorizer: 'org.apache.cassandra.auth.CassandraAuthorizer'
role_manager: CassandraRoleManager
roles_validity_in_ms: 2000
permissions_validity_in_ms: 2000
credentials_validity_in_ms: 2000
partitioner: org.apache.cassandra.dht.Murmur3Partitioner`)}

	runtimeBuilder.GenerateCM(configmapData)

	CassandraDeployment := GenerateDefaultCassandra(
		dcs,
	)

	return GenerateDefaultCassandraTestCase(
		t,
		vaultImpl,
		testName,
		CassandraDeployment,
		runtimeBuilder,
		nameSpace,
		nameSpaceRequestName,
	)
}

func TestExecutionCheck(t *testing.T) {
	testFuncs := []func() CaseStruct{
		func() CaseStruct {
			cs := GenerateDefaultCassandraWrapper(
				t,
				nil,
				"Two DCs All Services",
				3,
				2,
			)
			cs.executor.SetExecutable(cs.builder.Build(cs.ctx))
			cs.RunTestFunc = func() error {
				return cs.executor.Execute(cs.ctx)
			}
			return cs
		},

		func() CaseStruct {
			cs := GenerateDefaultCassandraWrapper(
				t,
				nil,
				"Two DCs ",
				3,
				2,
			)
			msS := cs.ctx.Get(constants.ContextSpec).(*v1alpha1.CassandraDeployment)
			msS.Spec.Cassandra.DeploymentSchema.DataCenters[0].Deploy = false
			msS.Spec.Cassandra.DeploymentSchema.DataCenters[1].Deploy = true
			cs.ctx.Set(constants.ContextSpec, msS)
			cs.executor.SetExecutable(cs.builder.Build(cs.ctx))
			cs.RunTestFunc = func() error {
				return cs.executor.Execute(cs.ctx)
			}
			// TODO services
			// cs.ReadResultFunc = func(t *testing.T, err error) {
			// 	robot := &v1app.Deployment{}
			// 	cl := cs.ctx.Get(constants.ContextClient).(client.Client)

			// 	err = cl.Get(context.TODO(),
			// 		types.NamespacedName{Name: utils.Robot, Namespace: cs.nameSpace}, robot)
			// 	if err != nil {
			// 		t.Error(err)
			// 	}

			// 	for _, env := range robot.Spec.Template.Spec.Containers[0].Env {
			// 		if env.Name == "DC_NAME" {
			// 			assert.Equal(t, msS.Spec.Cassandra.DeploymentSchema.DataCenters[1].Name, env.Value)

			// 		}
			// 	}
			// }
			return cs
		},
		func() CaseStruct {
			cs := GenerateDefaultCassandraWrapper(
				t,
				nil,
				"Two DCs with removed nodes",
				3,
				2,
			)
			msS := cs.ctx.Get(constants.ContextSpec).(*v1alpha1.CassandraDeployment)
			msS.Spec.Cassandra.DeploymentSchema.DataCenters[0].RemoveNodes = []map[string]string{{"1": "1234-5678"}}
			msS.Spec.Cassandra.DeploymentSchema.DataCenters[0].Seeds = 2
			cs.ctx.Set(constants.ContextSpec, msS)
			cs.executor.SetExecutable(cs.builder.Build(cs.ctx))
			cs.RunTestFunc = func() error {
				return cs.executor.Execute(cs.ctx)
			}
			cs.ReadResultFunc = func(t *testing.T, err error) {
				seedList := cs.ctx.Get(utils.CassandraSeeds).(string)
				assert.Equal(t, "cassandra0-0.cassandra.$(NAMESPACE).svc.cluster.local, cassandra2-0.cassandra.$(NAMESPACE).svc.cluster.local, cassandra3-0.cassandra.$(NAMESPACE).svc.cluster.local",
					seedList, "Seeds do not match")

				statefulsets := &v1app.StatefulSetList{}
				cl := cs.ctx.Get(constants.ContextClient).(client.Client)

				listOps := []client.ListOption{
					client.InNamespace(cs.nameSpace),
					client.MatchingLabelsSelector{labels.SelectorFromSet(map[string]string{"service": "cassandra-cluster"})},
				}

				err = cl.List(context.TODO(), statefulsets, listOps...)
				if err != nil {
					t.Error(err)
				}
				var names []string
				for _, statefulset := range statefulsets.Items {
					names = append(names, statefulset.Name)
				}

				assert.Equal(t, []string{"cassandra0", "cassandra2", "cassandra3", "cassandra4", "cassandra5"}, names, "Nope")

			}
			return cs
		},
		func() CaseStruct {
			cs := GenerateDefaultCassandraWrapper(
				t,
				nil,
				"One DC All Services One Storage",
				3,
				1,
			)
			cs.executor.SetExecutable(cs.builder.Build(cs.ctx))
			msS := cs.ctx.Get(constants.ContextSpec).(*v1alpha1.CassandraDeployment)
			msS.Spec.Cassandra.DeploymentSchema.DataCenters[0].Storage = []*mTypes.StorageRequirements{
				msS.Spec.Cassandra.DeploymentSchema.DataCenters[0].Storage[0],
			}
			cs.ctx.Set(constants.ContextSpec, msS)
			cs.RunTestFunc = func() error {
				return cs.executor.Execute(cs.ctx)
			}
			return cs
		},
		func() CaseStruct {
			cs := GenerateDefaultCassandraWrapper(
				t,
				nil,
				"Test main storage bad mount settings",
				3,
				2,
			)
			cs.executor.SetExecutable(cs.builder.Build(cs.ctx))
			msS := cs.ctx.Get(constants.ContextSpec).(*v1alpha1.CassandraDeployment)
			msS.Spec.Cassandra.DeploymentSchema.DataCenters[1].Storage[0].MountSettings.MountPath =
				msS.Spec.Cassandra.DeploymentSchema.DataCenters[1].Storage[1].MountSettings.MountPath
			cs.ctx.Set(constants.ContextSpec, msS)
			cs.ReadResultFunc = func(t *testing.T, err error) {
				t.Error()
			}
			cs.ReadErrorFunc = func(t *testing.T, err error) error {
				assert.EqualError(t, err, "dcName1 datacenter's first storage element mount settings are overridden")
				return nil
			}
			cs.RunTestFunc = func() error {
				return cs.executor.Execute(cs.ctx)
			}
			return cs
		},
		func() CaseStruct {
			cs := GenerateDefaultCassandraWrapper(
				t,
				nil,
				"Test overlapping storage mount settings",
				3,
				2,
			)
			cs.executor.SetExecutable(cs.builder.Build(cs.ctx))
			msS := cs.ctx.Get(constants.ContextSpec).(*v1alpha1.CassandraDeployment)
			msS.Spec.Cassandra.DeploymentSchema.DataCenters[1].Storage[1].MountSettings.MountPath =
				msS.Spec.Cassandra.DeploymentSchema.DataCenters[1].Storage[0].MountSettings.MountPath
			cs.ctx.Set(constants.ContextSpec, msS)
			cs.ReadResultFunc = func(t *testing.T, err error) {
				t.Error()
			}
			cs.ReadErrorFunc = func(t *testing.T, err error) error {
				assert.EqualError(t, err, "dcName1 datacenter's storage elements mount settings are overlapping")
				return nil
			}
			cs.RunTestFunc = func() error {
				return cs.executor.Execute(cs.ctx)
			}
			return cs
		},
		func() CaseStruct {
			cs := GenerateDefaultCassandraWrapper(
				t,
				nil,
				"Test empty additional storage mount settings",
				3,
				2,
			)
			cs.executor.SetExecutable(cs.builder.Build(cs.ctx))
			msS := cs.ctx.Get(constants.ContextSpec).(*v1alpha1.CassandraDeployment)
			msS.Spec.Cassandra.DeploymentSchema.DataCenters[0].Storage[1].MountSettings = nil
			cs.ctx.Set(constants.ContextSpec, msS)
			cs.ReadResultFunc = func(t *testing.T, err error) {
				t.Error()
			}
			cs.ReadErrorFunc = func(t *testing.T, err error) error {
				assert.EqualError(t, err, "Check dcName0 datacenter's storage mount settings. Name or MountPath is empty")
				return nil
			}
			cs.RunTestFunc = func() error {
				return cs.executor.Execute(cs.ctx)
			}
			return cs
		},
		func() CaseStruct {
			cs := GenerateDefaultCassandraWrapper(
				t,
				nil,
				"Test empty main storage mount settings",
				3,
				2,
			)
			cs.executor.SetExecutable(cs.builder.Build(cs.ctx))
			msS := cs.ctx.Get(constants.ContextSpec).(*v1alpha1.CassandraDeployment)
			msS.Spec.Cassandra.DeploymentSchema.DataCenters[0].Storage[0].MountSettings = nil
			cs.ctx.Set(constants.ContextSpec, msS)
			cs.ReadResultFunc = func(t *testing.T, err error) {
				t.Error()
			}
			cs.ReadErrorFunc = func(t *testing.T, err error) error {
				assert.EqualError(t, err, "dcName0 datacenter's first storage element mount settings are overridden")
				return nil
			}
			cs.RunTestFunc = func() error {
				return cs.executor.Execute(cs.ctx)
			}
			return cs
		},
		func() CaseStruct {
			cs := GenerateDefaultCassandraWrapper(
				t,
				nil,
				"Check cassandra-configuration configmap updated with complex structure",
				3,
				1,
			)
			cs.executor.SetExecutable(&cassandra.CassandraConfigurationUpgradeStep{}) // test only one step
			msS := cs.ctx.Get(constants.ContextSpec).(*v1alpha1.CassandraDeployment)
			msS.Spec.Configuration = `num_tokens: 500
data_file_directories:
  - /var/lib/test_directory/data`
			cs.ctx.Set(constants.ContextSpec, msS)

			cs.ReadResultFunc = func(t *testing.T, err error) {
				configMap := &v1core.ConfigMap{}
				client := cs.ctx.Get(constants.ContextClient).(client.Client)
				err = client.Get(context.TODO(),
					types.NamespacedName{Name: utils.CassandraConfiguration, Namespace: cs.nameSpace}, configMap)
				if err != nil {
					t.Error(err)
				}
				configMapFromCloud := make(map[interface{}]interface{})
				err = yaml.Unmarshal([]byte(configMap.Data["config"]), &configMapFromCloud)
				if configMapFromCloud["num_tokens"] != 500 ||
					configMapFromCloud["data_file_directories"].([]interface{})[0] != "/var/lib/test_directory/data" {
					t.Error()
				}
			}
			cs.RunTestFunc = func() error {
				return cs.executor.Execute(cs.ctx)
			}
			return cs
		}, func() CaseStruct {
			cs := GenerateDefaultCassandraWrapper(
				t,
				nil,
				"Check cassandra-configuration configmap updated",
				3,
				1,
			)
			cs.executor.SetExecutable(&cassandra.CassandraConfigurationUpgradeStep{}) // test only one step
			msS := cs.ctx.Get(constants.ContextSpec).(*v1alpha1.CassandraDeployment)
			msS.Spec.Configuration = `num_tokens: 500`
			cs.ctx.Set(constants.ContextSpec, msS)

			cs.ReadResultFunc = func(t *testing.T, err error) {
				configMap := &v1core.ConfigMap{}
				client := cs.ctx.Get(constants.ContextClient).(client.Client)
				err = client.Get(context.TODO(),
					types.NamespacedName{Name: utils.CassandraConfiguration, Namespace: cs.nameSpace}, configMap)
				if err != nil {
					t.Error(err)
				}
				configMapFromCloud := make(map[interface{}]interface{})
				err = yaml.Unmarshal([]byte(configMap.Data["config"]), &configMapFromCloud)
				if configMapFromCloud["num_tokens"] != 500 {
					t.Error()
				}
			}
			cs.RunTestFunc = func() error {
				return cs.executor.Execute(cs.ctx)
			}

			return cs
		}, func() CaseStruct {
			cs := GenerateDefaultCassandraWrapper(
				t,
				nil,
				"Check configmap add new pair",
				3,
				1,
			)
			cs.executor.SetExecutable(&cassandra.CassandraConfigurationUpgradeStep{}) // test only one step
			msS := cs.ctx.Get(constants.ContextSpec).(*v1alpha1.CassandraDeployment)
			msS.Spec.Configuration = `foo: bar`
			cs.ctx.Set(constants.ContextSpec, msS)

			cs.ReadResultFunc = func(t *testing.T, err error) {
				configMap := &v1core.ConfigMap{}
				client := cs.ctx.Get(constants.ContextClient).(client.Client)
				err = client.Get(context.TODO(),
					types.NamespacedName{Name: utils.CassandraConfiguration, Namespace: cs.nameSpace}, configMap)
				if err != nil {
					t.Error(err)
				}
				configMapFromCloud := make(map[interface{}]interface{})
				err = yaml.Unmarshal([]byte(configMap.Data["config"]), &configMapFromCloud)
				if configMapFromCloud["foo"] != "bar" {
					t.Error()
				}
			}
			cs.RunTestFunc = func() error {
				return cs.executor.Execute(cs.ctx)
			}
			return cs
		}, func() CaseStruct {
			cs := GenerateDefaultCassandraWrapper(
				t,
				nil,
				"Test Cassandra Service has correct labels",
				3,
				1,
			)
			cs.executor.SetExecutable(&cassandra.CassandraServicesStep{})
			cs.ReadResultFunc = func(t *testing.T, err error) {
				service := &v1core.Service{}
				client := cs.ctx.Get(constants.ContextClient).(client.Client)

				err = client.Get(context.TODO(),
					types.NamespacedName{Name: "cassandra", Namespace: cs.nameSpace}, service)
				if err != nil {
					t.Error(err)
				}
				shouldBeSelector1 := map[string]string{"service": "cassandra-cluster"}
				assert.Equal(t, shouldBeSelector1, service.Spec.Selector, "Different selectors")

				dcService := &v1core.Service{}
				err = client.Get(context.TODO(),
					types.NamespacedName{Name: "cassandra-dc-dcName0", Namespace: cs.nameSpace}, dcService)
				if err != nil {
					t.Error(err)
				}
				shouldBeSelector2 := map[string]string{"app": "cassandra-dc-dcName0"}
				assert.Equal(t, shouldBeSelector2, dcService.Spec.Selector, "Different selectors")

			}
			cs.RunTestFunc = func() error {
				return cs.executor.Execute(cs.ctx)
			}
			return cs
		},
		func() CaseStruct {
			cs := GenerateDefaultCassandraWrapper(
				t,
				nil,
				"Test Exact Seed list",
				3,
				1,
			)
			cs.executor.SetExecutable(cs.builder.Build(cs.ctx))
			msS := cs.ctx.Get(constants.ContextSpec).(*v1alpha1.CassandraDeployment)
			msS.Spec.Cassandra.DeploymentSchema.DataCenters[0].SeedList = []string{"192.168.101.21", "192.168.101.21", "192.168.101.21"}
			cs.ctx.Set(constants.ContextSpec, msS)
			cs.ReadResultFunc = func(t *testing.T, err error) {
				seedList := cs.ctx.Get(utils.CassandraSeeds).(string)
				assert.Equal(t, "192.168.101.21, 192.168.101.21, 192.168.101.21", seedList, "Seeds do not match")
			}
			cs.RunTestFunc = func() error {
				return cs.executor.Execute(cs.ctx)
			}
			return cs
		},
		func() CaseStruct {
			cs := GenerateDefaultCassandraWrapper(
				t,
				nil,
				"Test Exact Seed list when One DC deploy = false",
				3,
				2,
			)
			cs.executor.SetExecutable(cs.builder.Build(cs.ctx))
			msS := cs.ctx.Get(constants.ContextSpec).(*v1alpha1.CassandraDeployment)
			msS.Spec.Cassandra.DeploymentSchema.DataCenters[1].Deploy = false
			msS.Spec.Cassandra.DeploymentSchema.DataCenters[1].ClusterDomain = "cluster-2.local"
			msS.Spec.Cassandra.DeploymentSchema.DataCenters[0].SeedList = []string{"192.168.101.21", "192.168.101.22", "192.168.101.23"}
			msS.Spec.Cassandra.DeploymentSchema.DataCenters[1].SeedList = []string{"192.168.101.31", "192.168.101.32", "192.168.101.33"}
			cs.ctx.Set(constants.ContextSpec, msS)
			cs.ReadResultFunc = func(t *testing.T, err error) {
				seedList := cs.ctx.Get(utils.CassandraSeeds).(string)
				assert.Equal(t, "192.168.101.21, 192.168.101.22, 192.168.101.23", seedList, "Seeds do not match")
			}
			cs.RunTestFunc = func() error {
				return cs.executor.Execute(cs.ctx)
			}
			return cs
		},
		func() CaseStruct {
			cs := GenerateDefaultCassandraWrapper(
				t,
				nil,
				"Test hostNetwork enabled",
				3,
				1,
			)
			cs.executor.SetExecutable(cs.builder.Build(cs.ctx))
			msS := cs.ctx.Get(constants.ContextSpec).(*v1alpha1.CassandraDeployment)
			msS.Spec.Cassandra.HostNetwork = true
			cs.ctx.Set(constants.ContextSpec, msS)
			cs.ReadResultFunc = func(t *testing.T, err error) {
				statefulset := &v1app.StatefulSet{}
				client := cs.ctx.Get(constants.ContextClient).(client.Client)

				err = client.Get(context.TODO(),
					types.NamespacedName{Name: "cassandra0", Namespace: cs.nameSpace}, statefulset)
				if err != nil {
					t.Error(err)
				}
				assert.Equal(t, true, statefulset.Spec.Template.Spec.HostNetwork, "HostNetwork not propagated")
			}
			cs.RunTestFunc = func() error {
				return cs.executor.Execute(cs.ctx)
			}
			return cs
		},
		// TODO services
		// func() CaseStruct {
		// 	vaultImpl := mVault.FakeVaultHelper{}
		// 	vaultImpl.On("CheckSecretExists", mock.Anything).Return(false, make(map[string]interface{}), nil)
		// 	vaultImpl.On("GeneratePassword", mock.Anything).Return(mock.Anything, nil)
		// 	vaultImpl.On("StorePassword", mock.Anything, mock.Anything).Return(nil)
		// 	vaultImpl.On("IsDatabaseConfigExist", mock.Anything).Return(false, nil)
		// 	vaultImpl.On("CreateDatabaseConfig", mock.Anything, mock.Anything).Return(nil)
		// 	vaultImpl.On("CreateStaticRole", mock.Anything, mock.Anything).Return(nil)
		// 	vaultImpl.On("IsStaticRoleExists", mock.Anything).Return(true, nil)
		// 	roleMap := map[string]interface{}{"password": "12345"}
		// 	vaultImpl.On("GetStaticRoleCredentials", mock.Anything).Return(roleMap, nil).Times(2)
		// 	cs := GenerateDefaultCassandraWrapper(
		// 		t,
		// 		vaultImpl,
		// 		"Test PodSpec update with Vault init container",
		// 		3,
		// 		1,
		// 	)
		// 	msS := cs.ctx.Get(constants.ContextSpec).(*v1alpha1.CassandraDeployment)
		// 	msS.Spec.VaultRegistration.Enabled = true
		// 	msS.Spec.VaultDBEngine.Enabled = true
		// 	msS.Spec.VaultRegistration.Path = "secret"
		// 	cs.ctx.Set(constants.ContextSpec, msS)
		// 	cs.executor.SetExecutable(cs.builder.Build(cs.ctx))
		// 	cs.ReadResultFunc = func(t *testing.T, err error) {
		// 		backup := &v1app.Deployment{}
		// 		client := cs.ctx.Get(constants.ContextClient).(client.Client)

		// 		err = client.Get(context.TODO(),
		// 			types.NamespacedName{Name: utils.BackupDaemon, Namespace: cs.nameSpace}, backup)
		// 		if err != nil {
		// 			t.Error(err)
		// 		}

		// 		assert.Equal(t, 1, len(backup.Spec.Template.Spec.InitContainers))
		// 		assert.Equal(t, []string{"/vault/vault-env"}, backup.Spec.Template.Spec.Containers[0].Command)

		// 		dbaas := &v1app.Deployment{}
		// 		err = client.Get(context.TODO(),
		// 			types.NamespacedName{Name: utils.DbaasName, Namespace: cs.nameSpace}, dbaas)
		// 		if err != nil {
		// 			t.Error(err)
		// 		}

		// 		assert.Equal(t, 1, len(dbaas.Spec.Template.Spec.InitContainers))
		// 		assert.Equal(t, []string{"/vault/vault-env"}, dbaas.Spec.Template.Spec.Containers[0].Command)
		// 		assert.Equal(t, []string{"/usr/local/bin/entrypoint"}, dbaas.Spec.Template.Spec.Containers[0].Args)

		// 		monitoringAgent := &v1app.Deployment{}
		// 		err = client.Get(context.TODO(),
		// 			types.NamespacedName{Name: utils.MonitoringAgent, Namespace: cs.nameSpace}, monitoringAgent)
		// 		if err != nil {
		// 			t.Error(err)
		// 		}

		// 		assert.Equal(t, 1, len(monitoringAgent.Spec.Template.Spec.InitContainers))
		// 		assert.Equal(t, []string{"/vault/vault-env"}, monitoringAgent.Spec.Template.Spec.Containers[0].Command)
		// 		assert.Equal(t, []string{"telegraf"}, monitoringAgent.Spec.Template.Spec.Containers[0].Args)

		// 		robotTest := &v1app.Deployment{}
		// 		err = client.Get(context.TODO(),
		// 			types.NamespacedName{Name: utils.Robot, Namespace: cs.nameSpace}, robotTest)
		// 		if err != nil {
		// 			t.Error(err)
		// 		}
		// 		assert.Equal(t, 1, len(robotTest.Spec.Template.Spec.InitContainers))
		// 		assert.Equal(t, []string{"/vault/vault-env"}, robotTest.Spec.Template.Spec.Containers[0].Command)
		// 		assert.Equal(t, []string{"/docker-entrypoint.sh", "run-robot"}, robotTest.Spec.Template.Spec.Containers[0].Args)
		// 	}
		// 	cs.RunTestFunc = func() error {
		// 		return cs.executor.Execute(cs.ctx)
		// 	}
		// 	return cs
		// },
		func() CaseStruct {
			cs := GenerateDefaultCassandraWrapper(
				t,
				nil,
				"Check secret read from kubernetes",
				3,
				1,
			)
			msS := cs.ctx.Get(constants.ContextSpec).(*v1alpha1.CassandraDeployment)
			msS.Spec.VaultRegistration.Enabled = false
			cs.ctx.Set(constants.ContextSpec, msS)
			cs.executor.SetExecutable(cs.builder.Build(cs.ctx))
			cs.ReadResultFunc = func(t *testing.T, err error) {
				password := cs.ctx.Get(utils.ContextPasswordKey).(string)
				assert.Equal(t, "admin", password)
			}
			cs.RunTestFunc = func() error {
				return cs.executor.Execute(cs.ctx)
			}
			return cs
		},
		// func() CaseStruct {
		// 	cs := GenerateDefaultCassandraWrapper(
		// 		nil,
		// 		"EmptyDir check",
		// 		1,
		// 		1,
		// 	)

		// 	msS := cs.ctx.Get(constants.ContextSpec).(*v1alpha1.CassandraDeployment)
		// 	msS.Spec.Backup.Storage.EmptyDir = true
		// 	cs.executor.SetExecutable(cs.builder.Build(cs.ctx))
		// 	cs.RunTestFunc = func() error {
		// 		return cs.executor.Execute(cs.ctx)
		// 	}
		// 	cs.ReadResultFunc = func(t *testing.T, err error) {
		// 		backup := &v1app.Deployment{}
		// 		client := cs.ctx.Get(constants.ContextClient).(client.Client)

		// 		err = client.Get(context.TODO(),
		// 			types.NamespacedName{Name: utils.BackupDaemon, Namespace: cs.nameSpace}, backup)
		// 		if err != nil {
		// 			t.Error(err)
		// 		}

		// 		assert.True(t, backup.Spec.Template.Spec.Volumes[0].EmptyDir != nil)
		// 	}
		// 	return cs
		// },
	}

	tests := []CaseStruct{}
	for _, tf := range testFuncs {
		tests = append(tests, tf())
	}
	for _, tt := range tests {
		if tt.name == "Two DCs All Services" {
			continue
		}
		t.Run(tt.name, func(t *testing.T) {
			for key, elem := range tt.ctxToReplaceAfterServiceBuilt {
				tt.ctx.Set(key, elem)
			}
			err := tt.RunTestFunc()
			if err == nil {
				tt.ReadResultFunc(t, err)
			} else {
				err := err
				if tt.ReadErrorFunc != nil {
					// Error should be handled here or returned as unknown
					err = tt.ReadErrorFunc(t, err)
				}
				if err != nil {
					panic(err)
				}
			}

		})
	}
}
