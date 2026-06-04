package backup

import (
	"fmt"
	"strconv"
	"strings"

	v1 "github.com/Netcracker/qubership-cassandra-supplementary/api/v1alpha1"
	"github.com/Netcracker/qubership-cassandra-supplementary/pkg/utils"
	"github.com/Netcracker/qubership-nosqldb-operator-core/pkg/constants"
	"github.com/Netcracker/qubership-nosqldb-operator-core/pkg/core"
	coreUtils "github.com/Netcracker/qubership-nosqldb-operator-core/pkg/utils"
	"go.uber.org/zap"
	v12 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type LegacyBackupDeployment struct {
	core.DefaultExecutable
}

func (r *LegacyBackupDeployment) Execute(ctx core.ExecutionContext) error {
	request := ctx.Get(constants.ContextRequest).(reconcile.Request)
	spec := ctx.Get(constants.ContextSpec).(*v1.CassandraSupplService)
	backup := spec.Spec.Backup
	helperImpl := ctx.Get(utils.KubernetesHelperImpl).(core.KubernetesHelper)
	log := ctx.Get(constants.ContextLogger).(*zap.Logger)
	credsManager := ctx.Get(utils.ContextCredsManager).(utils.CredsManagerI)

	dcs := spec.Spec.Cassandra.DeploymentSchema.DataCenters
	var hosts []string

	secretVolumes := map[string]string{
		spec.Spec.Cassandra.SecretName: "/var/run/secrets/cassandra",
		backup.SecretName:              "/var/run/secrets/backup",
		utils.SSHSecret:                "/var/run/secrets/ssh",
	}

	if spec.Spec.AWSKeyspaces.Install {
		secretVolumes[spec.Spec.AWSKeyspaces.SecretName] = "/var/run/secrets/aws"
	}

	if backup.S3.Enabled {
		secretVolumes[backup.S3.SecretName] = "/var/run/secrets/s3"
	}

	var volumes []v12.Volume
	var volumeMounts []v12.VolumeMount

	secretVolumeMode := int32(256)

	for secretName, mountPath := range secretVolumes {

		volumeName := utils.SanitizeName(secretName)

		volumes = append(volumes, v12.Volume{
			Name: volumeName,
			VolumeSource: v12.VolumeSource{
				Secret: &v12.SecretVolumeSource{
					SecretName:  secretName,
					DefaultMode: &secretVolumeMode,
				},
			},
		})

		volumeMounts = append(volumeMounts, v12.VolumeMount{
			Name:      volumeName,
			MountPath: mountPath,
			ReadOnly:  true,
		})
	}

	for _, dc := range dcs {
		for replica := 0; replica < dc.Replicas; replica++ {
			hosts = append(hosts, fmt.Sprintf("cassandra%d-0.cassandra.%s.svc.cluster.local", replica, request.Namespace))
		}
	}

	// Environment variable  Start
	var envs []v12.EnvVar

	if spec.Spec.AWSKeyspaces.Install {
		envs = append(envs,
			coreUtils.GetPlainTextEnvVar("EXTERNAL_RESTORE", "true"),
			coreUtils.GetPlainTextEnvVar("AWS_RESTORE", "true"),
		)

	} else {
		//CASSANDRA_MAJOR_VERSION
		cm, err := helperImpl.GetConfigMap("cassandra-major-version", request.Namespace)
		if err != nil {
			return err
		}
		envs = append(envs,
			coreUtils.GetPlainTextEnvVar("CASSANDRA_HOSTS", strings.Join(hosts[:], " ")),
			coreUtils.GetPlainTextEnvVar("BACKUP_SCHEDULE", backup.BackupSchedule),
			coreUtils.GetPlainTextEnvVar("GRANULAR_SCHEDULE", backup.GranularBackupSchedule),
			coreUtils.GetPlainTextEnvVar("SCHEDULED_DBS", strings.Join(backup.GranularBackupScheduledDbs[:], ",")),
			coreUtils.GetPlainTextEnvVar("EVICTION_POLICY", backup.EvictionPolicy),
			coreUtils.GetPlainTextEnvVar("GRANULAR_EVICTION_POLICY", backup.GranularEvictionPolicy),
			coreUtils.GetPlainTextEnvVar("STORAGE", backup.StorageDirectory),
			coreUtils.GetPlainTextEnvVar("CASSANDRA_MAJOR_VERSION", cm.Data["majorVersion"]),
			coreUtils.GetPlainTextEnvVar("CONNECT_TIMEOUT", fmt.Sprint(spec.Spec.GocqlConnectTimeout)),
			coreUtils.GetPlainTextEnvVar("REQUEST_TIMEOUT", fmt.Sprint(spec.Spec.GocqlTimeout)),
		)
		if backup.S3.Enabled {
			envs = append(envs,
				coreUtils.GetPlainTextEnvVar("S3_ENABLED", strconv.FormatBool(backup.S3.Enabled)),
				coreUtils.GetPlainTextEnvVar("S3_BUCKET", backup.S3.BucketName),
				coreUtils.GetPlainTextEnvVar("S3_URL", backup.S3.EndpointUrl),
			)
			if backup.S3.SslVerify {
				envs = append(envs, coreUtils.GetPlainTextEnvVar("S3_CERTS_PATH", "/s3Certs"))

			}
		}
	}

	if spec.Spec.IpV6 {
		envs = append(envs, coreUtils.GetPlainTextEnvVar("BROADCAST_ADDRESS", "::"))
	}

	nodeSelector := map[string]string{}
	var pvcName string
	if !backup.Storage.EmptyDir {
		pvcName = ctx.Get(fmt.Sprintf(utils.BackupPvcName, 0)).([]string)[0]
		nodeLabels := ctx.Get(fmt.Sprintf(utils.PVNodesFormat, 0)).([]map[string]string)

		if len(nodeLabels) > 0 {
			nodeSelector = nodeLabels[0]
		}
	}

	dc := LegacyBackupDeploymentTemplate(
		pvcName,
		request.Namespace,
		spec.Spec.Backup.DockerImage,
		nodeSelector,
		*spec.Spec.Backup.Resources,
		envs,
		backup.StorageDirectory,
		backup.Storage.EmptyDir,
		utils.GetHTTPPort(spec.Spec.TLS.Enabled),
		utils.GetUriScheme(spec.Spec.TLS.Enabled),
		volumeMounts,
		volumes)

	err := credsManager.AddCredHashToPodTemplate([]string{spec.Spec.Cassandra.SecretName}, &dc.Spec.Template)
	if err != nil {
		log.Error(fmt.Sprintf("can't add secret HASH to annotations for %s", dc.Name), zap.Error(err))
		return err
	}

	if backup.S3.SslVerify {

		dc.Spec.Template.Spec.Volumes = append(dc.Spec.Template.Spec.Volumes,
			v12.Volume{
				Name: "s3-ssl-certs",
				VolumeSource: v12.VolumeSource{
					Secret: &v12.SecretVolumeSource{
						SecretName: backup.S3.SslSecretName,
					},
				},
			},
		)

		dc.Spec.Template.Spec.Containers[0].VolumeMounts = append(dc.Spec.Template.Spec.Containers[0].VolumeMounts,
			v12.VolumeMount{
				Name:      "s3-ssl-certs",
				ReadOnly:  true,
				MountPath: "/s3Certs",
			},
		)
	}

	coreUtils.VaultPodSpec(&dc.Spec.Template.Spec, utils.BackupEntrypoint, spec.Spec.VaultRegistration)
	utils.TLSClientSpecUpdate(&dc.Spec.Template.Spec, utils.RootCertPath, spec.Spec.TLS)
	utils.TLSServerSpecUpdate(&dc.Spec.Template.Spec, spec.Spec.TLS, spec.Spec.Backup.TLS.BackupDaemonCASecretName, utils.ServerCertsPath)

	err = helperImpl.DeleteDeploymentAndPods(dc.Name, request.Namespace, spec.Spec.WaitTimeout)

	if err != nil {
		return err
	}

	labels := utils.BasicLabels{
		AppName:              utils.BackupDaemon,
		AppComponent:         "backend",
		AppTechnology:        "python",
		AppPartOf:            "cassandra-services",
		AppManagedBy:         "operator",
		AppManagedByOperator: "cassandra-services-operator",
	}
	err = utils.CreateRuntimeObjectContextWrapper(ctx, dc, dc.ObjectMeta, labels)

	core.PanicError(err, log.Error, "Error happened on processing backup deployment config")

	log.Debug("Waiting for backup is ready")
	err = helperImpl.WaitForPodsReady(
		map[string]string{
			utils.Name: utils.BackupDaemon,
		},
		request.Namespace,
		1,
		spec.Spec.WaitTimeout)

	core.PanicError(err, log.Error, "Pods waiting failed")

	return nil
}

func (r *LegacyBackupDeployment) Condition(ctx core.ExecutionContext) (bool, error) {
	return true, nil
}
