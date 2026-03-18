package cassandra

import (
	"fmt"
	"regexp"

	"github.com/Netcracker/qubership-cassandra-operator/api/v1alpha1"
	"github.com/Netcracker/qubership-cassandra-operator/pkg/impl/utils"
	"github.com/Netcracker/qubership-nosqldb-operator-core/pkg/constants"
	"github.com/Netcracker/qubership-nosqldb-operator-core/pkg/consul"
	"github.com/Netcracker/qubership-nosqldb-operator-core/pkg/core"
	"github.com/Netcracker/qubership-nosqldb-operator-core/pkg/types"
	"github.com/hashicorp/consul/api"
	"go.uber.org/zap"
	v1 "k8s.io/api/apps/v1"
	v13 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func CassandraReplicaTemplate(
	cassandraEnvs []v13.EnvVar,
	reaperContainer v13.Container,
	dockerImage string,
	podSecurityContext *v13.PodSecurityContext,
	replicaNumber int,
	namespace string,
	nodeSelector map[string]string,
	tolerations []v13.Toleration,
	resources v13.ResourceRequirements,
	dataCenterName string,
	livnessProbeRetries int32,
	pvcMounts map[string]*v13.VolumeMount,
	hostNework bool,
	spec *v1alpha1.CassandraDeployment) *v1.StatefulSet {

	tls := spec.Spec.TLS
	var volumeMounts []v13.VolumeMount
	var volumes []v13.Volume

	for pvcName, volumeMount := range pvcMounts {
		volumeMounts = append(volumeMounts, *volumeMount)

		volumes = append(volumes, v13.Volume{
			Name: volumeMount.Name,
			VolumeSource: v13.VolumeSource{
				PersistentVolumeClaim: &v13.PersistentVolumeClaimVolumeSource{
					ClaimName: pvcName,
				},
			},
		})
	}

	dcServiceName := fmt.Sprintf(utils.CassandraDCFormat, dataCenterName)

	reg, err := regexp.Compile(`([0-9]+\.[0-9]+\.[0-9]+$)`)
	if err != nil {
		panic(err)
	}
	labels := map[string]string{
		utils.Service:              utils.CassandraCluster,
		utils.App:                  dcServiceName,
		utils.ReplicaNumber:        fmt.Sprintf("%v", replicaNumber),
		utils.AppPartOf:            spec.Spec.PartOf,
		utils.AppManagedBy:         spec.Spec.ManagedBy,
		utils.CloneModeType:        "data",
		utils.AppVersion:           reg.FindString(dockerImage),
		utils.AppManagedByOperator: "cassandra-operator",
	}

	name := fmt.Sprintf(utils.CassandraReplicaNameFormat, replicaNumber)

	podLabels := labels
	podLabels[utils.Name] = name

	var replicas int32 = 1
	var tgps int64 = 10

	var containers []v13.Container
	allowPrivilegeEscalation := false

	containers = append(containers, v13.Container{
		Name:            name,
		Image:           dockerImage,
		ImagePullPolicy: spec.Spec.ImagePullPolicy,
		SecurityContext: &v13.SecurityContext{
			Capabilities: &v13.Capabilities{
				Drop: []v13.Capability{"ALL"},
			},
			AllowPrivilegeEscalation: &allowPrivilegeEscalation,
		},
		Command: []string{
			"/bin/bash", "-c", "rm -f /var/lib/cassandra/data/system/peer*/*.*; /run.sh",
		},
		Ports: []v13.ContainerPort{
			{
				Name:          "intra-node",
				ContainerPort: 7000,
			},
			{
				Name:          "tls-intra-node",
				ContainerPort: 7001,
			},
			{
				Name:          "jmx",
				ContainerPort: 7199,
			},
			{
				Name:          "cql",
				ContainerPort: 9042,
			},
			{
				Name:          "tcp-upd-port",
				ContainerPort: 8778,
			},
		},
		Resources: resources,
		Lifecycle: &v13.Lifecycle{
			PreStop: &v13.LifecycleHandler{
				Exec: &v13.ExecAction{
					Command: []string{
						"/bin/bash", "-c", "nodetool flush",
					},
				},
			},
		},
		Env: cassandraEnvs,
		LivenessProbe: &v13.Probe{
			ProbeHandler: v13.ProbeHandler{
				TCPSocket: &v13.TCPSocketAction{
					Port: intstr.IntOrString{Type: intstr.Int, IntVal: 9042},
				},
			},
			InitialDelaySeconds: 100,
			TimeoutSeconds:      10,
			PeriodSeconds:       20,
			SuccessThreshold:    1,
			FailureThreshold:    livnessProbeRetries,
		},
		ReadinessProbe: &v13.Probe{
			ProbeHandler: v13.ProbeHandler{
				TCPSocket: &v13.TCPSocketAction{
					Port: intstr.IntOrString{Type: intstr.Int, IntVal: 9042},
				},
			},
			InitialDelaySeconds: 100,
			TimeoutSeconds:      10,
			PeriodSeconds:       20,
			SuccessThreshold:    1,
			FailureThreshold:    3,
		},
		VolumeMounts: append(volumeMounts,
			v13.VolumeMount{
				Name:      utils.Configuration,
				MountPath: utils.ConfigurationPath,
			}),
	})

	volProj := []v13.VolumeProjection{
		{
			ConfigMap: &v13.ConfigMapProjection{
				LocalObjectReference: v13.LocalObjectReference{
					Name: utils.CassandraConfiguration,
				},
				Items: []v13.KeyToPath{
					{
						Path: "cassandra.yaml",
						Key:  utils.Config,
					},
				},
			},
		},
		{
			ConfigMap: &v13.ConfigMapProjection{
				LocalObjectReference: v13.LocalObjectReference{
					Name: utils.CassandraEnv,
				},
				Items: []v13.KeyToPath{
					{
						Path: "cassandra-env.sh",
						Key:  utils.Config,
					},
				},
			},
		},
		{
			ConfigMap: &v13.ConfigMapProjection{
				LocalObjectReference: v13.LocalObjectReference{
					Name: utils.CassandraJVM,
				},
				Items: []v13.KeyToPath{
					{
						Path: "jvm-server.options",
						Key:  utils.Config,
					},
				},
			},
		},
		{
			ConfigMap: &v13.ConfigMapProjection{
				LocalObjectReference: v13.LocalObjectReference{
					Name: utils.CassandraLogback,
				},
				Items: []v13.KeyToPath{
					{
						Path: "logback.xml",
						Key:  utils.Config,
					},
				},
			},
		},
	}

	if spec.Spec.Cassandra.CommitlogArchiving.Enabled {
		volProj = append(volProj, v13.VolumeProjection{
			ConfigMap: &v13.ConfigMapProjection{
				LocalObjectReference: v13.LocalObjectReference{
					Name: utils.CassandraCommitlogArchiving,
				},
				Items: []v13.KeyToPath{
					{
						Path: "commitlog_archiving.properties",
						Key:  utils.Config,
					},
				},
			},
		})
	}

	if tls.Enabled {
		volProj = append(volProj,
			v13.VolumeProjection{
				Secret: &v13.SecretProjection{
					LocalObjectReference: v13.LocalObjectReference{
						Name: tls.RootCASecretName,
					},
					Items: []v13.KeyToPath{
						{
							Path: tls.RootCAFileName,
							Key:  tls.RootCAFileName,
						},
						{
							Path: tls.PrivateKeyFileName,
							Key:  tls.PrivateKeyFileName,
						},
						{
							Path: tls.SignedCRTFileName,
							Key:  tls.SignedCRTFileName,
						},
					},
				},
			},
		)
	}

	if reaperContainer.Name != "" {

		vm := v13.VolumeMount{
			Name:      "reaper",
			MountPath: "/etc/cassandra-reaper-temp/",
			ReadOnly:  false,
		}
		reaperContainer.VolumeMounts = append(reaperContainer.VolumeMounts, vm)
		volumeProjection := v13.VolumeProjection{
			ConfigMap: &v13.ConfigMapProjection{
				LocalObjectReference: v13.LocalObjectReference{
					Name: utils.CassandraReaper,
				},
				Items: []v13.KeyToPath{
					{
						Path: "cassandra-reaper.yml",
						Key:  utils.Config,
					},
				},
			},
		}
		volumeSource := v13.VolumeSource{
			Projected: &v13.ProjectedVolumeSource{
				Sources: []v13.VolumeProjection{volumeProjection},
			},
		}
		volumes = append(volumes, v13.Volume{
			Name:         "reaper",
			VolumeSource: volumeSource,
		})

		if tls.Enabled {
			vm := v13.VolumeMount{
				Name:      "reaper-tls",
				MountPath: "/usr/ssl/",
			}
			reaperContainer.VolumeMounts = append(reaperContainer.VolumeMounts, vm)
			volumeProjection := v13.VolumeProjection{
				Secret: &v13.SecretProjection{
					LocalObjectReference: v13.LocalObjectReference{
						Name: tls.RootCASecretName,
					},
					Items: []v13.KeyToPath{
						{
							Path: tls.RootCAFileName,
							Key:  tls.RootCAFileName,
						},
						{
							Path: tls.PrivateKeyFileName,
							Key:  tls.PrivateKeyFileName,
						},
						{
							Path: tls.SignedCRTFileName,
							Key:  tls.SignedCRTFileName,
						},
					},
				},
			}
			volumeSource := v13.VolumeSource{
				Projected: &v13.ProjectedVolumeSource{
					Sources: []v13.VolumeProjection{volumeProjection},
				},
			}
			volumes = append(volumes, v13.Volume{
				Name:         "reaper-tls",
				VolumeSource: volumeSource,
			})
		}
		containers = append(containers, reaperContainer)
	}
	// Default anti-affinity rule (used only if user didn't provide one)
	defaultAffinity := &v13.Affinity{
		PodAntiAffinity: &v13.PodAntiAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: []v13.PodAffinityTerm{
				{
					LabelSelector: &v12.LabelSelector{
						MatchExpressions: []v12.LabelSelectorRequirement{
							{
								Key:      utils.Service,
								Operator: v12.LabelSelectorOpIn,
								Values:   []string{utils.CassandraCluster},
							},
						},
					},
					TopologyKey: utils.KubeHostName,
				},
			},
		},
	}

	// Use user-defined affinity if available; otherwise use default
	var affinity *v13.Affinity
	if spec.Spec.Cassandra.Affinity != nil {
		affinity = spec.Spec.Cassandra.Affinity
	} else {
		affinity = defaultAffinity
	}

	return &v1.StatefulSet{
		ObjectMeta: v12.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: v1.StatefulSetSpec{
			ServiceName: utils.Cassandra,
			Replicas:    &replicas,
			Selector: &v12.LabelSelector{
				MatchLabels: labels,
			},
			Template: v13.PodTemplateSpec{
				ObjectMeta: v12.ObjectMeta{
					Labels: podLabels,
				},
				Spec: v13.PodSpec{
					TerminationGracePeriodSeconds: &tgps,
					Affinity:                      affinity,
					HostNetwork:                   hostNework,
					PriorityClassName:             spec.Spec.PriorityClassName,
					ServiceAccountName:            spec.Spec.ServiceAccountName,
					NodeSelector:                  nodeSelector,
					Tolerations:                   tolerations,
					SecurityContext:               podSecurityContext,
					Containers:                    containers,
					Volumes: append(volumes, []v13.Volume{
						{
							Name: utils.Configuration,
							VolumeSource: v13.VolumeSource{
								Projected: &v13.ProjectedVolumeSource{
									Sources:     volProj,
									DefaultMode: nil,
								},
							},
						},
					}...),
				},
			},
		},
	}

}

func CassandraConsulServiceRegistrationCast(ctx core.ExecutionContext, client consul.ConsulClient, registration *types.AgentServiceRegistration, logger *zap.Logger) *api.AgentServiceRegistration {
	spec := ctx.Get(constants.ContextSpec).(*v1alpha1.CassandraDeployment)
	request := ctx.Get(constants.ContextRequest).(reconcile.Request)

	address := utils.CalcServiceHostName(true, nil, request.Namespace, utils.DefaultClusterDomain)

	// Lock settings
	sr := &registration.AgentServiceRegistration
	sr.ID = sr.Name
	sr.Address = address
	sr.Port = 9042
	sr.EnableTagOverride = true

	// Set some useful info
	dataCenters := utils.FilterDC(spec.Spec.Cassandra.DeploymentSchema.DataCenters, func(dc *v1alpha1.DataCenter) bool { return dc.Deploy })

	commonMeta := map[string]string{
		"deployed-data-centers-count": fmt.Sprintf("%v", len(dataCenters)),
		"namespace":                   request.Namespace,
	}
	var commonTags []string
	for _, dc := range dataCenters {
		commonMeta[dc.Name+"-active-replicas"] = fmt.Sprintf("%v", dc.GetActiveReplicasLen())
		commonTags = append(commonTags, dc.Name)
	}

	sr.Tags = append(sr.Tags, commonTags...)
	sr.Meta = core.ConcatMaps(commonMeta, sr.Meta)

	if sr.Check == nil {
		panic("Service check for cassandra is not found")
	}

	check := *sr.Check
	check.Name = "tcp-check"
	check.TCP = fmt.Sprintf("%s:%v", sr.Address, sr.Port)

	sr.Checks = api.AgentServiceChecks{
		&check,
	}
	sr.Check = nil

	return sr
}
