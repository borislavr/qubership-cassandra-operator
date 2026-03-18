package dbaas

import (
	"fmt"
	"os"
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

type DbaasDeployment struct {
	core.DefaultExecutable
}

func (r *DbaasDeployment) Execute(ctx core.ExecutionContext) error {
	request := ctx.Get(constants.ContextRequest).(reconcile.Request)
	spec := ctx.Get(constants.ContextSpec).(*v1.CassandraSupplService)
	dbaas := spec.Spec.Dbaas
	helperImpl := ctx.Get(utils.KubernetesHelperImpl).(core.KubernetesHelper)
	log := ctx.Get(constants.ContextLogger).(*zap.Logger)
	credsManager := ctx.Get(utils.ContextCredsManager).(utils.CredsManagerI)
	tlsEnabled := utils.IsTLSEnableForDBAAS(spec.Spec.Dbaas.Aggregator.DbaasAggregatorRegistrationAddress, spec.Spec.TLS.Enabled)

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

	envs = append(envs,
		coreUtils.GetPlainTextEnvVar("CASSANDRA_HOSTNAME", core.OptionalString(spec.Spec.Cassandra.Host, fmt.Sprintf("%s.%s", utils.Cassandra, request.Namespace))),
		coreUtils.GetPlainTextEnvVar("CASSANDRA_PORT", core.OptionalString(strconv.Itoa(spec.Spec.Cassandra.Port), "9042")),
		coreUtils.GetSecretEnvVar("CASSANDRA_USERNAME", spec.Spec.Cassandra.SecretName, utils.Username),
		coreUtils.GetSecretEnvVar("CASSANDRA_PASSWORD", spec.Spec.Cassandra.SecretName, utils.Password),
		coreUtils.GetPlainTextEnvVar("GOCQL_DEFAULT_KEYSPACE", core.OptionalString(spec.Spec.Cassandra.DefaultKeyspace, "system")),
		coreUtils.GetPlainTextEnvVar("GOCQL_CONSISTENCY", core.OptionalString(spec.Spec.Cassandra.Consistency, "QUORUM")),
		coreUtils.GetPlainTextEnvVar("TLS_ENABLED", strconv.FormatBool(spec.Spec.Cassandra.TLS)),
		coreUtils.GetPlainTextEnvVar("DBAAS_AGGREGATOR_PHYSICAL_DATABASE_IDENTIFIER", core.OptionalString(dbaas.Aggregator.PhysicalDatabaseIdentifier, request.Namespace)),
		coreUtils.GetPlainTextEnvVar("DBAAS_ADAPTER_ADDRESS", fmt.Sprintf("%s://%s.%s:%d", utils.GetHTTPProtocol(tlsEnabled), utils.DbaasName, request.Namespace, utils.GetHTTPPort(tlsEnabled))),
		coreUtils.GetPlainTextEnvVar("DBAAS_AGGREGATOR_REGISTRATION_ADDRESS", dbaas.Aggregator.DbaasAggregatorRegistrationAddress),
		coreUtils.GetPlainTextEnvVar("PORT", fmt.Sprint(utils.GetHTTPPort(tlsEnabled))),
		coreUtils.GetSecretEnvVar("DBAAS_ADAPTER_USERNAME", dbaas.Adapter.SecretName, utils.Username),
		coreUtils.GetSecretEnvVar("DBAAS_ADAPTER_PASSWORD", dbaas.Adapter.SecretName, utils.Password),
		coreUtils.GetSecretEnvVar("DBAAS_AGGREGATOR_REGISTRATION_USERNAME", dbaas.Aggregator.SecretName, utils.Username),
		coreUtils.GetSecretEnvVar("DBAAS_AGGREGATOR_REGISTRATION_PASSWORD", dbaas.Aggregator.SecretName, utils.Password),
		coreUtils.GetPlainTextEnvVar("GOCQL_TIMEOUT", fmt.Sprint(spec.Spec.GocqlTimeout)),
		coreUtils.GetPlainTextEnvVar("GOCQL_CONNECT_TIMEOUT", fmt.Sprint(spec.Spec.GocqlConnectTimeout)),
		coreUtils.GetPlainTextEnvVar("VAULT_ENABLED", strconv.FormatBool(spec.Spec.VaultRegistration.Enabled)),
		coreUtils.GetPlainTextEnvVar("CLOUD_PUBLIC_HOST", os.Getenv("CLOUD_PUBLIC_HOST")),
		coreUtils.GetPlainTextEnvVar("VAULT_AUTH_METHOD", spec.Spec.VaultRegistration.Method),
		coreUtils.GetPlainTextEnvVar("VAULT_ENV_PASSTHROUGH", "VAULT_ADDR,VAULT_ROLE,VAULT_AUTH_METHOD,VAULT_ENABLED"),
		coreUtils.GetPlainTextEnvVar("VAULT_ROTATION_PERIOD", strconv.Itoa(spec.Spec.VaultRegistration.RotationPeriod)),
		coreUtils.GetPlainTextEnvVar("VAULT_DB_ENGINE_NAME", "cassandra-db-engine"), // TODO hardcoded
		coreUtils.GetPlainTextEnvVar("API_VERSION", dbaas.ApiVersion),
		coreUtils.GetPlainTextEnvVar("MULTI_USERS_ENABLED", strconv.FormatBool(dbaas.MultiUsers)),
		coreUtils.GetPlainTextEnvVar("CASSANDRA_DEFAULT_TOPOLOGY", dbaas.TopologyStrategy),
		coreUtils.GetSecretEnvVar("DBAAS_STREAMING_ROLE_NAME", utils.DbaasAdminRoleCreds, utils.Name),
		coreUtils.GetSecretEnvVar("DBAAS_STREAMING_ROLE_PERMISSIONS", utils.DbaasAdminRoleCreds, utils.Roles),
	)

	if spec.Spec.Backup.Install {
		envs = append(envs,
			coreUtils.GetSecretEnvVar("BACKUP_DAEMON_API_CREDENTIALS_USERNAME", spec.Spec.Backup.SecretName, utils.Username),
			coreUtils.GetSecretEnvVar("BACKUP_DAEMON_API_CREDENTIALS_PASSWORD", spec.Spec.Backup.SecretName, utils.Password),
			coreUtils.GetPlainTextEnvVar("BACKUP_DAEMON_ADDRESS", fmt.Sprintf("%s://cassandra-backup-daemon:%d", utils.GetHTTPProtocol(spec.Spec.TLS.Enabled), utils.GetHTTPPort(spec.Spec.TLS.Enabled))),
		)
	}
	// Environment variable End

	dc := DbaasDeploymentTemplate(
		request.Namespace,
		dbaas.DockerImage,
		dbaas.NodeLabels,
		*dbaas.Resources,
		envs,
		utils.GetHTTPPort(tlsEnabled))

	err := credsManager.AddCredHashToPodTemplate([]string{spec.Spec.Cassandra.SecretName}, &dc.Spec.Template)
	if err != nil {
		log.Error(fmt.Sprintf("can't add secret HASH to annotations for %s", dc.Name), zap.Error(err))
		return err
	}

	coreUtils.VaultPodSpec(&dc.Spec.Template.Spec, []string{"/usr/local/bin/entrypoint"}, spec.Spec.VaultRegistration)
	utils.TLSClientSpecUpdate(&dc.Spec.Template.Spec, utils.RootCertPath, spec.Spec.TLS)

	if tlsEnabled {
		utils.TLSServerSpecUpdate(&dc.Spec.Template.Spec, spec.Spec.TLS, spec.Spec.Dbaas.TLS.DbaasAdapterCASecretName, utils.ServerCertsPath)
	}

	err = helperImpl.DeleteDeploymentAndPods(dc.Name, request.Namespace, spec.Spec.WaitTimeout)

	core.PanicError(err, log.Error, "Dbaas deployment deletion failed")

	labels := utils.BasicLabels{
		AppName:              utils.DbaasName,
		AppComponent:         "backend",
		AppTechnology:        "go",
		AppPartOf:            "cassandra-services",
		AppManagedBy:         "operator",
		AppManagedByOperator: "cassandra-services-operator",
	}

	err = utils.CreateRuntimeObjectContextWrapper(ctx, dc, dc.ObjectMeta, labels)
	core.PanicError(err, log.Error, "Dbaas deployment config processing failed")

	log.Debug("Waiting for dbaas is ready")
	err = helperImpl.WaitForPodsReady(
		map[string]string{
			utils.Name: utils.DbaasName,
		},
		request.Namespace,
		1,
		spec.Spec.WaitTimeout)

	core.PanicError(err, log.Error, "Dbaas Pod Ready status waiting failed")

	return nil
}
