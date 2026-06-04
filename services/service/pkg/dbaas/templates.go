package dbaas

import (
	"github.com/Netcracker/qubership-cassandra-supplementary/pkg/utils"
	v12 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func DbaasDeploymentTemplate(namespace string,
	image string,
	nodeSelector map[string]string,
	resources v1.ResourceRequirements,
	env []v1.EnvVar,
	port int32,
	volumeMounts []v1.VolumeMount,
	volumes []v1.Volume) *v12.Deployment {

	var replicas int32 = 1
	allowPrivilegeEscalation := false
	readOnlyRootFilesystem := true
	dc := &v12.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      utils.DbaasName,
			Namespace: namespace,
			Labels: map[string]string{
				utils.App:          utils.CassandraCluster,
				utils.Microservice: utils.DbaasName,
				utils.Name:         utils.DbaasName,
				utils.AppPartOf:    "cassandra-services",
				utils.AppManagedBy: "operator",
			},
		},
		Spec: v12.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					utils.Name: utils.DbaasName,
				},
			},
			Replicas: &replicas,
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
					Labels: map[string]string{
						utils.Name: utils.DbaasName,
					},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:  utils.DbaasName,
							Image: image,
							SecurityContext: &v1.SecurityContext{
								Capabilities: &v1.Capabilities{
									Drop: []v1.Capability{"ALL"},
								},
								AllowPrivilegeEscalation: &allowPrivilegeEscalation,
								ReadOnlyRootFilesystem:   &readOnlyRootFilesystem,
							},
							Ports: []v1.ContainerPort{
								{
									Name:          "http",
									ContainerPort: port,
									Protocol:      "TCP",
								},
							},
							Env:       env,
							Resources: resources,
							VolumeMounts: append(
								[]v1.VolumeMount{
									{
										Name:      "dbaas-physical-databases-labels-mount",
										MountPath: "/app/config",
									},
								},
								volumeMounts...,
							),
							LivenessProbe: &v1.Probe{
								ProbeHandler: v1.ProbeHandler{
									TCPSocket: &v1.TCPSocketAction{
										Port: intstr.IntOrString{Type: intstr.Int, IntVal: port},
									},
								},
								InitialDelaySeconds: 5,
								TimeoutSeconds:      5,
								PeriodSeconds:       7,
								SuccessThreshold:    1,
								FailureThreshold:    12,
							},
							ReadinessProbe: &v1.Probe{
								ProbeHandler: v1.ProbeHandler{
									TCPSocket: &v1.TCPSocketAction{
										Port: intstr.IntOrString{Type: intstr.Int, IntVal: port},
									},
								},
								InitialDelaySeconds: 5,
								TimeoutSeconds:      5,
								PeriodSeconds:       7,
								SuccessThreshold:    1,
								FailureThreshold:    12,
							},
						},
					},
					NodeSelector: nodeSelector,
					Volumes: append(
						[]v1.Volume{
							{
								Name: "dbaas-physical-databases-labels-mount",
								VolumeSource: v1.VolumeSource{
									ConfigMap: &v1.ConfigMapVolumeSource{
										LocalObjectReference: v1.LocalObjectReference{
											Name: "nc-dbaas-physical-databases-labels",
										},
										DefaultMode: func() *int32 {
											mode := int32(420)
											return &mode
										}(),
									},
								},
							},
						},
						volumes...,
					),
				},
			},
		},
	}
	return dc
}
