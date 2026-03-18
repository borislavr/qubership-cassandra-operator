package main

import (
	"context"
	"fmt"
	"testing"

	cqlMocks "github.com/Netcracker/qubership-cql-driver/mocks"
	v1 "github.com/Netcracker/qubership-cassandra-supplementary/api/v1alpha1"
	"github.com/Netcracker/qubership-cassandra-supplementary/pkg"
	"github.com/Netcracker/qubership-cassandra-supplementary/pkg/utils"
	"github.com/Netcracker/qubership-nosqldb-operator-core/pkg/constants"
	"github.com/Netcracker/qubership-nosqldb-operator-core/pkg/core"
	mTypes "github.com/Netcracker/qubership-nosqldb-operator-core/pkg/types"
	mVault "github.com/Netcracker/qubership-nosqldb-operator-core/pkg/vault"
	"github.com/Netcracker/qubership-nosqldb-operator-core/pkg/vault/mocks"
	"go.uber.org/zap"

	"github.com/hashicorp/vault/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	v1app "k8s.io/api/apps/v1"
	v1core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type FakeVaultImpl struct {
	t        *testing.T
	password string
}

func (r *FakeVaultImpl) GetClient() *api.Client {
	return nil
}

func (r *FakeVaultImpl) VaultGeneratePasswordWithPolicy(policyName string, client *api.Client) (string, error) {
	return r.password, nil
}

func (r *FakeVaultImpl) VaultCreatePasswordPolicy(policyName string, policy string, client *api.Client) error {
	return nil
}
func (r *FakeVaultImpl) VaultRead(path string, client *api.Client) (map[string]interface{}, error) {
	return nil, nil
}

func (r *FakeVaultImpl) VaultWrite(path string, secret map[string]interface{}, client *api.Client) error {
	return nil
}

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

type MockCredsManager struct {
}

func (c *MockCredsManager) AddCredHashToPodTemplate(secretNames []string, template *v1core.PodTemplateSpec) error {
	return nil
}

func generateSecrets(namespace string, secretName string, user string, pass string) *v1core.Secret {
	return &v1core.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
		},
		Data: map[string][]byte{"username": []byte(user), "password": []byte(pass)},
	}
}

func generatePV(nameFormat string, nodeFormat string, namespace string, size int) ([]runtime.Object, []string, []map[string]string) {
	pvS := []runtime.Object{}
	names := []string{}
	nodeLabels := []map[string]string{}
	for i := 1; i <= size; i++ {
		pvS = append(pvS, &v1core.PersistentVolume{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf(nameFormat, i),
				Namespace: namespace,
				Labels: map[string]string{
					"node": fmt.Sprintf(nodeFormat, i),
				},
			},
		})
		names = append(names, fmt.Sprintf(nameFormat, i))
		nodeLabels = append(nodeLabels, map[string]string{utils.KubeHostName: fmt.Sprintf(nodeFormat, i)})
	}

	return pvS, names, nodeLabels
}
func generateDCPV(nameFormat string, nodeFormat string, namespace string, dcIndex, size int) ([]runtime.Object, []string, []map[string]string) {
	pvS := []runtime.Object{}
	names := []string{}
	nodeLabels := []map[string]string{}
	for i := 0; i < size; i++ {
		index := dcIndex*size + i
		pvS = append(pvS, &v1core.PersistentVolume{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf(nameFormat, dcIndex, i),
				Namespace: namespace,
				Labels: map[string]string{
					utils.KubeHostName: fmt.Sprintf(nodeFormat, index),
				},
			},
		})
		names = append(names, fmt.Sprintf(nameFormat, dcIndex, i))
		nodeLabels = append(nodeLabels, map[string]string{utils.KubeHostName: fmt.Sprintf(nodeFormat, index)})
	}

	return pvS, names, nodeLabels
}

func GenerateDefaultCassandra(namespace string, cassandraDCs []*v1.DataCenter, backupPvName []string, backupNodeLabels []map[string]string) *v1.CassandraSupplService {
	GiQuantity, _ := resource.ParseQuantity("5Gi")
	var fsGroup int64 = 999
	var tolerationSeconds int64 = 20

	rr := &v1core.ResourceRequirements{
		Limits: v1core.ResourceList{
			v1core.ResourceMemory: GiQuantity,
		},
		Requests: nil,
	}

	return &v1.CassandraSupplService{
		Spec: v1.CassandraServiceSpec{
			WaitTimeout: 100,
			Recycler: mTypes.Recycler{
				Resources: rr,
			},
			ServiceAccountName: "cassandra-operator",
			Policies: &v1.Policies{
				Tolerations: []v1core.Toleration{
					{
						Key:               "key1",
						Value:             "value1",
						Operator:          v1core.TolerationOpEqual,
						Effect:            v1core.TaintEffectNoSchedule,
						TolerationSeconds: &tolerationSeconds,
					},
					{
						Key:               "key2",
						Value:             "value2",
						Operator:          v1core.TolerationOpEqual,
						Effect:            v1core.TaintEffectNoExecute,
						TolerationSeconds: &tolerationSeconds,
					},
				},
			},
			PodSecurityContext: &v1core.PodSecurityContext{
				FSGroup: &fsGroup,
			},
			Cassandra: v1.Cassandra{
				Install:          true,
				User:             "admin",
				SecretName:       "cassandra-secret",
				DeploymentSchema: &v1.DeploymentSchema{DataCenters: cassandraDCs},
			},

			Backup: v1.Backup{
				Install:    true,
				User:       "backup",
				SecretName: "cassandra-backup-api-credentials",
				Storage: &mTypes.StorageRequirements{
					Size:       []string{"5Gi"},
					Volumes:    backupPvName,
					NodeLabels: backupNodeLabels,
				},
				Resources: rr,
				S3: v1.S3backup{
					Enabled:         false,
					SecretName:      "cassandra-backup-s3-credentials",
					BucketName:      "product_backups_development",
					AccessKeyId:     "AccessKeyId",
					AccessKeySecret: "accessKeySecret",
					EndpointUrl:     "https://storage.googleapis.com",
				},
			},
			Dbaas: v1.Dbaas{
				Install:   true,
				Resources: rr,
				Adapter: &v1.DbaasAdapterCredentials{
					Username:   "adapter",
					SecretName: "dbaas-adapter-credentials",
				},
				Aggregator: &v1.DbaasAggregatorCredentials{
					Username:                   "agg",
					PhysicalDatabaseIdentifier: "ind",
					SecretName:                 "dbaas-aggregator-credentials",
				},
			},
			Monitoring: v1.Monitoring{
				Install: true,
			},
			RobotTests: v1.RobotTests{
				Install:           true,
				Resources:         rr,
				ReplicationFactor: 3,
				Args: []string{
					"run-robot",
				},
			},
			VaultRegistration: mTypes.VaultRegistration{
				InitContainerResources: rr,
			},
		},
	}
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
	vaultImpl mVault.VaultHelper,
	testName string,
	cassandraSupplService *v1.CassandraSupplService,
	runtimeObjects []runtime.Object,
	nameSpace string,
	nameSpaceRequestName string,
) CaseStruct {
	client := fake.NewFakeClient(runtimeObjects...)
	utilsHelp := &TestUtilsImpl{}
	utilsHelp.ForceKey = true
	// Because there is empty runtime Scheme
	utilsHelp.OwnerKey = false
	utilsHelp.Client = client

	clusterBuilder := &cqlMocks.ClusterBuilder{}
	clusterBuilder.On("WithHost", mock.Anything).Return(clusterBuilder)
	clusterBuilder.On("WithUser", mock.Anything).Return(clusterBuilder)
	clusterBuilder.On("WithPassword", mock.Anything).Return(clusterBuilder)
	clusterBuilder.On("WithRootCertPath", mock.Anything).Return(clusterBuilder)
	clusterBuilder.On("WithTLSEnabled", mock.Anything).Return(clusterBuilder)
	clusterBuilder.On("WithKeyspace", mock.Anything).Return(clusterBuilder)
	clusterBuilder.On("WithConsistency", mock.Anything).Return(clusterBuilder)
	clusterBuilder.On("Build", mock.Anything).Return(&cqlMocks.TestCluster{})

	caseStruct := CaseStruct{
		name:      testName,
		nameSpace: nameSpace,
		executor:  core.DefaultExecutor(),
		builder:   &pkg.CassandraServiceBuilder{},
		ctx: core.GetExecutionContext(map[string]interface{}{
			constants.ContextSpec:   cassandraSupplService,
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
			"contextResourceOwner":                 cassandraSupplService, //todo hardcode replace
			constants.ContextServiceDeploymentInfo: map[string]string{},
			constants.ContextVault:                 vaultImpl,
			constants.ContextHashConfigMap:         "random",
			utils.ContextClusterBuilder:            clusterBuilder,
		}),
		ctxToReplaceAfterServiceBuilt: map[string]interface{}{
			utils.KubernetesHelperImpl:  utilsHelp,
			utils.ContextClusterBuilder: clusterBuilder,
			utils.ContextCredsManager:   &MockCredsManager{},
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

func configConfigMap(namespace string) []runtime.Object {
	return []runtime.Object{&v1core.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      "cassandra-major-version",
		},
		BinaryData: map[string][]byte{
			"Data": []byte("majorVersion: '4'")},
	}}
}

func GenerateDefaultCassandraWrapper(
	vaultImpl mVault.VaultHelper,
	testName string,
	replicas int,
	dcCount int,
) CaseStruct {

	nameSpace := "cassandra-namespace"
	nameSpaceRequestName := "cassandra-name"

	backupPv, backupName, backupNodes := generatePV(utils.BackupPvcName, "node-%v", nameSpaceRequestName, 1)

	secret := generateSecrets(nameSpace, "cassandra-secret", "admin", "admin")

	allPvs := []runtime.Object{}
	dcs := []*v1.DataCenter{}

	for i := 0; i < dcCount; i++ {
		dcs = append(dcs,
			&v1.DataCenter{
				Name:     fmt.Sprintf("dcName%v", i),
				Replicas: replicas,
				Deploy:   true,
			},
		)

	}

	runtimeObjects := append(allPvs, backupPv...)
	runtimeObjects = append(runtimeObjects, secret)
	runtimeObjects = append(configConfigMap(nameSpace), secret)

	cassandraService := GenerateDefaultCassandra(
		nameSpace,
		dcs,
		backupName,
		backupNodes,
	)

	return GenerateDefaultCassandraTestCase(
		vaultImpl,
		testName,
		cassandraService,
		runtimeObjects,
		nameSpace,
		nameSpaceRequestName,
	)
}

func TestExecutionCheck(t *testing.T) {
	testFuncs := []func() CaseStruct{
		func() CaseStruct {
			cs := GenerateDefaultCassandraWrapper(
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
			vaultImpl := &mocks.FakeVaultHelper{}
			vaultImpl.On("CheckSecretExists", mock.Anything).Return(false, make(map[string]interface{}), nil)
			vaultImpl.On("GeneratePassword", mock.Anything).Return(mock.Anything, nil)
			vaultImpl.On("StorePassword", mock.Anything, mock.Anything).Return(nil)
			vaultImpl.On("IsDatabaseConfigExist", mock.Anything).Return(false, nil)
			vaultImpl.On("CreateDatabaseConfig", mock.Anything, mock.Anything).Return(nil)
			vaultImpl.On("CreateStaticRole", mock.Anything, mock.Anything).Return(nil)
			vaultImpl.On("IsStaticRoleExists", mock.Anything).Return(true, nil)
			vaultImpl.On("IsStaticRoleExists", mock.Anything).Return(true, nil)
			vaultImpl.On("ResolvePassword", mock.Anything).Return("12345", nil)
			roleMap := map[string]interface{}{"password": "12345"}

			vaultImpl.On("GetStaticRoleCredentials", mock.Anything).Return(roleMap, nil).Times(2)
			cs := GenerateDefaultCassandraWrapper(
				vaultImpl,
				"Test PodSpec update with Vault init container",
				3,
				1,
			)
			msS := cs.ctx.Get(constants.ContextSpec).(*v1.CassandraSupplService)
			msS.Spec.VaultRegistration.Enabled = true
			msS.Spec.VaultRegistration.Path = "secret"
			msS.Spec.Monitoring.Install = true
			msS.Spec.Monitoring.MetricCollector = "influxDB"
			msS.Spec.Dbaas.Install = true
			msS.Spec.RobotTests.Install = true
			cs.ctx.Set(constants.ContextSpec, msS)
			cs.executor.SetExecutable(cs.builder.Build(cs.ctx))
			cs.ReadResultFunc = func(t *testing.T, err error) {
				backup := &v1app.Deployment{}
				client := cs.ctx.Get(constants.ContextClient).(client.Client)

				err = client.Get(context.TODO(),
					types.NamespacedName{Name: utils.BackupDaemon, Namespace: cs.nameSpace}, backup)
				if err != nil {
					t.Error(err)
				}

				assert.Equal(t, 1, len(backup.Spec.Template.Spec.InitContainers))
				assert.Equal(t, []string{"/vault/vault-env"}, backup.Spec.Template.Spec.Containers[0].Command)

				dbaas := &v1app.Deployment{}
				err = client.Get(context.TODO(),
					types.NamespacedName{Name: utils.DbaasName, Namespace: cs.nameSpace}, dbaas)
				if err != nil {
					t.Error(err)
				}

				assert.Equal(t, 1, len(dbaas.Spec.Template.Spec.InitContainers))
				assert.Equal(t, []string{"/vault/vault-env"}, dbaas.Spec.Template.Spec.Containers[0].Command)
				assert.Equal(t, []string{"/usr/local/bin/entrypoint"}, dbaas.Spec.Template.Spec.Containers[0].Args)

				robotTest := &v1app.Deployment{}
				err = client.Get(context.TODO(),
					types.NamespacedName{Name: utils.Robot, Namespace: cs.nameSpace}, robotTest)
				if err != nil {
					t.Error(err)
				}
				assert.Equal(t, 1, len(robotTest.Spec.Template.Spec.InitContainers))
				assert.Equal(t, []string{"/vault/vault-env"}, robotTest.Spec.Template.Spec.Containers[0].Command)
				assert.Equal(t, []string{"/docker-entrypoint.sh", "run-robot"}, robotTest.Spec.Template.Spec.Containers[0].Args)
			}
			cs.RunTestFunc = func() error {
				return cs.executor.Execute(cs.ctx)
			}
			return cs
		},
		// func() CaseStruct {
		// 	cs := GenerateDefaultCassandraWrapper(
		// 		nil,
		// 		"Check secret read from kubernetes",
		// 		3,
		// 		1,
		// 	)
		// 	msS := cs.ctx.Get(constants.ContextSpec).(*v1.CassandraSupplService)
		// 	msS.Spec.VaultRegistration.Enabled = false
		// 	cs.ctx.Set(constants.ContextSpec, msS)
		// 	cs.executor.SetExecutable(cs.builder.Build(cs.ctx))
		// 	cs.ReadResultFunc = func(t *testing.T, err error) {
		// 		password := cs.ctx.Get(utils.ContextPasswordKey).(string)
		// 		assert.Equal(t, "admin", password)
		// 	}
		// 	cs.RunTestFunc = func() error {
		// 		return cs.executor.Execute(cs.ctx)
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
