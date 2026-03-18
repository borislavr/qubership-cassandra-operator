package robotTests

import (
	"github.com/Netcracker/qubership-cassandra-supplementary/pkg/utils"
	v12 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func RobotTemplate(namespace string,
	image string,
	resources v1.ResourceRequirements,
	nodeSelector map[string]string,
	env []v1.EnvVar,
	args []string) *v12.Deployment {

	allowPrivilegeEscalation := false
	var replicas int32 = 1
	dc := &v12.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      utils.Robot,
			Namespace: namespace,
			Labels: map[string]string{
				utils.App:          utils.CassandraCluster,
				utils.Microservice: utils.Robot,
				utils.Name:         utils.Robot,
			},
		},
		Spec: v12.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					utils.Name: utils.Robot,
				},
			},
			Replicas: &replicas,
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
					Labels: map[string]string{
						utils.Name: utils.Robot,
					},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						v1.Container{
							Name:      utils.Robot,
							Image:     image,
							Env:       env,
							Resources: resources,
							Args:      args,
							SecurityContext: &v1.SecurityContext{
								Capabilities: &v1.Capabilities{
									Drop: []v1.Capability{"ALL"},
								},
								AllowPrivilegeEscalation: &allowPrivilegeEscalation,
							},
						},
					},
					NodeSelector: nodeSelector,
				},
			},
		},
	}

	return dc
}
