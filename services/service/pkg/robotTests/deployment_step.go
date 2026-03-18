package robotTests

import (
	"fmt"
	"strconv"

	v1 "github.com/Netcracker/qubership-cassandra-supplementary/api/v1alpha1"
	"github.com/Netcracker/qubership-cassandra-supplementary/pkg/utils"
	"github.com/Netcracker/qubership-nosqldb-operator-core/pkg/constants"
	"github.com/Netcracker/qubership-nosqldb-operator-core/pkg/core"
	coreUtils "github.com/Netcracker/qubership-nosqldb-operator-core/pkg/utils"
	"go.uber.org/zap"
	v12 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type RobotDeployment struct {
	core.DefaultExecutable
}

func (r *RobotDeployment) Execute(ctx core.ExecutionContext) error {
	request := ctx.Get(constants.ContextRequest).(reconcile.Request)
	spec := ctx.Get(constants.ContextSpec).(*v1.CassandraSupplService)
	robot := spec.Spec.RobotTests
	helperImpl := ctx.Get(utils.KubernetesHelperImpl).(core.KubernetesHelper)
	credsManager := ctx.Get(utils.ContextCredsManager).(utils.CredsManagerI)
	log := ctx.Get(constants.ContextLogger).(*zap.Logger)

	// Environment variable Start
	envs := []v12.EnvVar{
		{
			Name: "NAMESPACE",
			ValueFrom: &v12.EnvVarSource{
				FieldRef: &v12.ObjectFieldSelector{
					FieldPath: "metadata.namespace",
				},
			},
		},
	}

	currentDc := utils.NewStream(spec.Spec.Cassandra.DeploymentSchema.DataCenters).FindFirst(func(dc interface{}) bool {
		return dc.(*v1.DataCenter).Deploy
	}).(*v1.DataCenter)

	envs = append(envs,
		coreUtils.GetPlainTextEnvVar("CASSANDRA_HOST", core.OptionalString(spec.Spec.Cassandra.Host, fmt.Sprintf("%s.%s", utils.Cassandra, request.Namespace))),
		coreUtils.GetPlainTextEnvVar("CASSANDRA_PORT", core.OptionalString(strconv.Itoa(spec.Spec.Cassandra.Port), "9042")),
		coreUtils.GetSecretEnvVar("CASSANDRA_USERNAME", spec.Spec.Cassandra.SecretName, utils.Username),
		coreUtils.GetSecretEnvVar("CASSANDRA_PASSWORD", spec.Spec.Cassandra.SecretName, utils.Password),
		coreUtils.GetPlainTextEnvVar("TEST_KEYSPACES_REPLICATION_FACTOR", strconv.Itoa(robot.ReplicationFactor)),
		coreUtils.GetPlainTextEnvVar("ATTEMPTS_NUMBER", strconv.Itoa(robot.AttemptsNumber)),
		coreUtils.GetPlainTextEnvVar("PROMETHEUS_URL", robot.PrometheusUrl),
		coreUtils.GetPlainTextEnvVar("TAGS", robot.Tags),
		coreUtils.GetPlainTextEnvVar("WAIT_TIMEOUT", strconv.Itoa(spec.Spec.WaitTimeout)),
		coreUtils.GetPlainTextEnvVar("DC_NAME", currentDc.Name),
		coreUtils.GetPlainTextEnvVar("DBAAS_ADAPTER_API_VERSION", spec.Spec.Dbaas.ApiVersion),
		coreUtils.GetPlainTextEnvVar("PORT", fmt.Sprint(utils.GetHTTPPort(spec.Spec.TLS.Enabled))),
		coreUtils.GetPlainTextEnvVar("CONFIG_NAME", "cassandra-tests-config"),
		coreUtils.GetPlainTextEnvVar("SUPPLEMENTARY_CONFIG_NAME", "supplementary-tests-config"),

		//todo better place or variables
		coreUtils.GetPlainTextEnvVar("STATUS_CUSTOM_RESOURCE_PATH", fmt.Sprintf("apps/v1/%s/deployments/robot-tests", request.Namespace)),
		coreUtils.GetPlainTextEnvVar("STATUS_WRITING_ENABLED", "true"),
	)

	if spec.Spec.Backup.Install {
		envs = append(envs,
			coreUtils.GetPlainTextEnvVar("BACKUP_HOST", fmt.Sprintf("%s.%s.svc", utils.BackupDaemon, request.Namespace)),
			coreUtils.GetSecretEnvVar("BACKUP_DAEMON_API_CREDENTIALS_USERNAME", spec.Spec.Backup.SecretName, utils.Username),
			coreUtils.GetSecretEnvVar("BACKUP_DAEMON_API_CREDENTIALS_PASSWORD", spec.Spec.Backup.SecretName, utils.Password),
		)
	}

	if spec.Spec.Dbaas.Install {
		envs = append(envs,
			coreUtils.GetPlainTextEnvVar("DBAAS_HOST", fmt.Sprintf("%s.%s.svc", utils.DbaasName, request.Namespace)),
			coreUtils.GetSecretEnvVar("DBAAS_ADAPTER_USERNAME", spec.Spec.Dbaas.Adapter.SecretName, utils.Username),
			coreUtils.GetSecretEnvVar("DBAAS_ADAPTER_PASSWORD", spec.Spec.Dbaas.Adapter.SecretName, utils.Password),
		)
	}

	// Environment variable End

	dc := RobotTemplate(
		request.Namespace,
		robot.DockerImage,
		*robot.Resources,
		robot.NodeLabels,
		envs,
		spec.Spec.RobotTests.Args)

	err := credsManager.AddCredHashToPodTemplate([]string{spec.Spec.Cassandra.SecretName}, &dc.Spec.Template)
	if err != nil {
		log.Error(fmt.Sprintf("can't add secret HASH to annotations for %s", dc.Name), zap.Error(err))
		return err
	}

	robotArgs := append(utils.RobotEntrypoint, spec.Spec.Args...)

	coreUtils.VaultPodSpec(&dc.Spec.Template.Spec, robotArgs, spec.Spec.VaultRegistration)
	utils.TLSClientSpecUpdate(&dc.Spec.Template.Spec, utils.RootCertPath, spec.Spec.TLS)

	err = helperImpl.DeleteDeploymentAndPods(dc.Name, request.Namespace, spec.Spec.WaitTimeout)

	core.PanicError(err, log.Error, "RobotTests deployment config processing failed")

	labels := utils.BasicLabels{
		AppName:       utils.Robot,
		AppComponent:  "operator",
		AppTechnology: "python",
	}

	err = utils.CreateRuntimeObjectContextWrapper(ctx, dc, dc.ObjectMeta, labels)
	core.PanicError(err, log.Error, "RobotTests deployment config processing failed")

	log.Debug("Waiting for robot tests ready")

	err = helperImpl.WaitForTestsReady(dc.Name, dc.Namespace, spec.Spec.WaitTimeout)

	core.PanicError(err, log.Error, "RobotTests failed")

	return nil
}
