package cassandra

import (
	"fmt"
	"strconv"

	"github.com/Netcracker/qubership-cassandra-operator/api/v1alpha1"
	"github.com/Netcracker/qubership-cassandra-operator/pkg/impl/utils"
	"github.com/Netcracker/qubership-nosqldb-operator-core/pkg/constants"
	"github.com/Netcracker/qubership-nosqldb-operator-core/pkg/core"
	coreUtils "github.com/Netcracker/qubership-nosqldb-operator-core/pkg/utils"
	"go.uber.org/zap"
	v12 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type CassandraStatefulSetStep struct {
	core.DefaultExecutable
}

func (r *CassandraStatefulSetStep) Execute(ctx core.ExecutionContext) error {
	request := ctx.Get(constants.ContextRequest).(reconcile.Request)
	log := ctx.Get(constants.ContextLogger).(*zap.Logger)
	spec := ctx.Get(constants.ContextSpec).(*v1alpha1.CassandraDeployment)
	helperImpl := ctx.Get(utils.KubernetesHelperImpl).(core.KubernetesHelper)
	client := ctx.Get(constants.ContextClient).(client.Client)

	log.Info("Cassandra Statefulsets creation started")
	var liveness int32 = 3 + core.MaxInt32(int32(spec.Spec.WaitTimeout), 10)/10
	cassandra := spec.Spec.Cassandra
	tls := spec.Spec.TLS

	dcReplicas := utils.FilterDC(spec.Spec.Cassandra.DeploymentSchema.DataCenters, func(dc *v1alpha1.DataCenter) bool { return dc.Deploy })
	dataCentersCount := len(dcReplicas)
	allowPrivilegeEscalation := false

	hostNetwork := false
	if spec.Spec.Cassandra.HostNetwork {
		hostNetwork = true
	}

	var reaperContainer v12.Container

	if spec.Spec.Reaper.Install {
		trustStore, err := core.ReadSecret(client, spec.Spec.Reaper.TruststoreSecretName, request.Namespace)
		if err != nil {
			log.Error(fmt.Sprintf("Failed to read secret %s, err: %v", spec.Spec.Reaper.TruststoreSecretName, err))
		}
		var reaperEnvs []v12.EnvVar
		reaperEnvs = append(reaperEnvs,
			coreUtils.GetSecretEnvVar("REAPER_CASS_AUTH_USERNAME", spec.Spec.Cassandra.SecretName, utils.Username),
			coreUtils.GetSecretEnvVar("REAPER_CASS_AUTH_PASSWORD", spec.Spec.Cassandra.SecretName, utils.Password),
			coreUtils.GetPlainTextEnvVar("REAPER_JMX_AUTH_USERNAME", "reaperUser"),
			coreUtils.GetPlainTextEnvVar("REAPER_JMX_AUTH_PASSWORD", "reaperPass"),
			coreUtils.GetPlainTextEnvVar("REAPER_STORAGE_TYPE", "cassandra"),
			coreUtils.GetPlainTextEnvVar("REAPER_CASS_CLUSTER_NAME", request.Namespace), //todo!
			coreUtils.GetPlainTextEnvVar("REAPER_CASS_CONTACT_POINTS", `[{"host":"127.0.0.1","port":9042}]`),
			coreUtils.GetPlainTextEnvVar("REAPER_CASS_KEYSPACE", "reaper_db"),
			coreUtils.GetPlainTextEnvVar("REAPER_CASS_AUTH_ENABLED", "true"),
			coreUtils.GetPlainTextEnvVar("REAPER_DATACENTER_AVAILABILITY", "SIDECAR"),
			coreUtils.GetSecretEnvVar("REAPER_AUTH_USER", spec.Spec.Reaper.SecretName, utils.Username),
			coreUtils.GetSecretEnvVar("REAPER_AUTH_PASSWORD", spec.Spec.Reaper.SecretName, utils.Password),
			coreUtils.GetPlainTextEnvVar("REAPER_CASS_NATIVE_PROTOCOL_SSL_ENCRYPTION_ENABLED", strconv.FormatBool(spec.Spec.TLS.Enabled)),
			coreUtils.GetPlainTextEnvVar("JAVA_OPTS", fmt.Sprintf("-Djavax.net.ssl.trustStore=/tmp/truststore.jks\n-Djavax.net.ssl.trustStorePassword=%s",
				trustStore.Data[utils.Password])),
			coreUtils.GetPlainTextEnvVar("REAPER_SERVER_TYPE", spec.Spec.Reaper.Type),
			coreUtils.GetPlainTextEnvVar("REAPER_SERVER_APP_PORT", strconv.Itoa(spec.Spec.Reaper.Port)),
		)

		for envKey, envValue := range spec.Spec.Reaper.Envs {
			reaperEnvs = append(reaperEnvs,
				coreUtils.GetPlainTextEnvVar(envKey, envValue),
			)
		}

		reaperContainer = v12.Container{
			Name:  "cassandra-reaper",
			Image: spec.Spec.Reaper.DockerImage,
			Ports: []v12.ContainerPort{
				{
					Name:          "web-ui",
					ContainerPort: int32(spec.Spec.Reaper.Port),
				},
				{
					Name:          "admin-ui",
					ContainerPort: 8081,
				},
			},
			Env:       reaperEnvs,
			Resources: *spec.Spec.Reaper.Resources,
			SecurityContext: &v12.SecurityContext{
				Capabilities: &v12.Capabilities{
					Drop: []v12.Capability{"ALL"},
				},
				AllowPrivilegeEscalation: &allowPrivilegeEscalation,
			},
		}
	}

	mQuantity, _ := resource.ParseQuantity("1Mi")

	for dc := 0; dc < dataCentersCount; dc++ {
		seeds := ctx.Get(utils.CassandraSeeds).(string)
		nodeLabels := ctx.Get(fmt.Sprintf(utils.PVNodesFormat, dc)).([]map[string]string)
		var cassandraEnvs []v12.EnvVar
		cassandraEnvs = append(cassandraEnvs,
			v12.EnvVar{
				Name: "NAMESPACE",
				ValueFrom: &v12.EnvVarSource{
					FieldRef: &v12.ObjectFieldSelector{
						FieldPath: "metadata.namespace",
					},
				},
			},
			v12.EnvVar{
				Name: "MEMORY_LIMIT",
				ValueFrom: &v12.EnvVarSource{
					ResourceFieldRef: &v12.ResourceFieldSelector{
						Resource: "limits.memory",
						Divisor:  mQuantity,
					},
				},
			},
			v12.EnvVar{
				Name: "CQLSH_HOST",
				ValueFrom: &v12.EnvVarSource{
					FieldRef: &v12.ObjectFieldSelector{
						FieldPath: "status.podIP",
					},
				},
			},
			v12.EnvVar{
				Name: "POD_IP",
				ValueFrom: &v12.EnvVarSource{
					FieldRef: &v12.ObjectFieldSelector{
						FieldPath: "status.podIP",
					},
				},
			},
			coreUtils.GetPlainTextEnvVar("CASSANDRA_SEEDS", seeds),
			coreUtils.GetPlainTextEnvVar("CASSANDRA_DC", dcReplicas[dc].Name),
			coreUtils.GetPlainTextEnvVar("RUN_AS_ROOT", "false"),
			coreUtils.GetPlainTextEnvVar("MEM_OVERHEAD", "128"),
		)

		if tls.Enabled {
			cassandraEnvs = append(cassandraEnvs,
				coreUtils.GetPlainTextEnvVar("TLS", "true"),
				coreUtils.GetPlainTextEnvVar("TLS_SIGNED", utils.ConfigurationPath+tls.SignedCRTFileName),
				coreUtils.GetPlainTextEnvVar("TLS_KEY", utils.ConfigurationPath+tls.PrivateKeyFileName),
				coreUtils.GetPlainTextEnvVar("TLS_CA", utils.ConfigurationPath+tls.RootCAFileName),
				coreUtils.GetPlainTextEnvVar("TLS_PASS", tls.KeystorePass),
			)
		}

		for envKey, envValue := range spec.Spec.Cassandra.Envs {
			cassandraEnvs = append(cassandraEnvs,
				coreUtils.GetPlainTextEnvVar(envKey, envValue),
			)
		}

		racks := dcReplicas[dc].Racks
		rackName := "rack1"

		broadcastAddressList := dcReplicas[dc].BroadcastAddress
		listenAddressList := dcReplicas[dc].ListenAddress
		rpcBroadcastAddressList := dcReplicas[dc].RpcBroadcastAddress
		rpcListenAddressList := dcReplicas[dc].RpcListenAddress

		for i, replicaIndex := range dcReplicas[dc].GetActiveReplicas() {
			nodeSelector := map[string]string{}
			if len(nodeLabels) > 0 {
				nodeSelector = nodeLabels[replicaIndex%len(nodeLabels)]
			}

			if len(racks) > 0 {
				rackName = racks[replicaIndex%len(racks)]
			}

			replicaEnvs := append(cassandraEnvs,
				coreUtils.GetPlainTextEnvVar("CASSANDRA_RACK", rackName),
			)

			if len(broadcastAddressList) > 0 {
				broadcastAddress := dcReplicas[dc].BroadcastAddress[replicaIndex]
				replicaEnvs = append(replicaEnvs,
					coreUtils.GetPlainTextEnvVar("CASSANDRA_BROADCAST_ADDRESS", broadcastAddress),
				)
			}

			if len(listenAddressList) > 0 {
				listenAddress := dcReplicas[dc].ListenAddress[replicaIndex]
				replicaEnvs = append(replicaEnvs,
					coreUtils.GetPlainTextEnvVar("CASSANDRA_LISTEN_ADDRESS", listenAddress),
				)
			}

			if len(rpcBroadcastAddressList) > 0 {
				rpcBroadcastAddress := dcReplicas[dc].RpcBroadcastAddress[replicaIndex]
				replicaEnvs = append(replicaEnvs,
					coreUtils.GetPlainTextEnvVar("CASSANDRA_BROADCAST_RPC_ADDRESS", rpcBroadcastAddress),
				)
			}

			if len(rpcListenAddressList) > 0 {
				rpcListenAddress := dcReplicas[dc].RpcListenAddress[replicaIndex]
				replicaEnvs = append(replicaEnvs,
					coreUtils.GetPlainTextEnvVar("CASSANDRA_RPC_ADDRESS", rpcListenAddress),
				)
			}

			pvcWithMountSettings := make(map[string]*v12.VolumeMount)
			for storageIndex, storage := range dcReplicas[dc].Storage {
				pvcContextFormat := fmt.Sprintf(utils.CassandraDCPvcNameFormat, dc)
				pvcNames := ctx.Get(fmt.Sprintf("%s-%v", pvcContextFormat, storageIndex)).([]string)
				pvcWithMountSettings[pvcNames[i%len(pvcNames)]] = storage.MountSettings
			}

			var tolerations []v12.Toleration
			if spec.Spec.Policies != nil {
				tolerations = spec.Spec.Policies.Tolerations
			}

			if spec.Spec.Reaper.Install {
				reaperContainer.Env = append(reaperContainer.Env, coreUtils.GetPlainTextEnvVar("REAPER_CASS_LOCAL_DC", dcReplicas[dc].Name))
			}

			ss := CassandraReplicaTemplate(
				replicaEnvs,
				reaperContainer,
				cassandra.DockerImage,
				spec.Spec.PodSecurityContext,
				utils.CalcReplicaIndex(dcReplicas, dc, replicaIndex),
				request.Namespace,
				nodeSelector,
				tolerations,
				*cassandra.Resources,
				dcReplicas[dc].Name,
				liveness,
				pvcWithMountSettings,
				hostNetwork,
				spec)

			ssPodLabels := ss.Spec.Template.ObjectMeta.Labels

			err := helperImpl.DeleteStatefulsetAndPods(ss.Name, request.Namespace, spec.Spec.WaitTimeout)

			core.PanicError(err, log.Error, "Cassandra statefulset "+ss.Name+" deletion failed")

			err = utils.CreateRuntimeObjectContextWrapper(ctx, ss, ss.ObjectMeta)
			core.PanicError(err, log.Error, "Cassandra dc"+strconv.Itoa(dc)+" "+strconv.Itoa(replicaIndex)+" statefulset failed")

			log.Debug(fmt.Sprintf("Statefulset %s has been created", ss.Name))

			err = helperImpl.WaitForPodsReady(
				ssPodLabels,
				request.Namespace,
				1,
				spec.Spec.WaitTimeout)

			core.PanicError(err, log.Error, "Pods waiting failed")

			log.Debug(fmt.Sprintf("Statefulset %s has been started", ss.Name))
		}
	}

	return nil
}

func (r *CassandraStatefulSetStep) Condition(ctx core.ExecutionContext) (bool, error) {
	return true, nil
}
