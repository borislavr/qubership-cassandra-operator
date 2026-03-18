package tests

import (
	"fmt"

	"github.com/Netcracker/qubership-cassandra-operator/pkg/impl/utils"
	v1core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type RuntimeObjectBuilder struct {
	namespace      string
	runtimeObjects []runtime.Object
}

func (r *RuntimeObjectBuilder) GenerateSecrets(secretName string, user string, pass string) *RuntimeObjectBuilder {
	r.runtimeObjects = append(r.runtimeObjects, &v1core.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: r.namespace,
		},
		Data: map[string][]byte{"username": []byte(user), "password": []byte(pass)},
	})

	return r
}

func (r *RuntimeObjectBuilder) GeneratePVs(nameFormat string, nodeFormat string, dcIndex, size int) ([]string, []map[string]string) {
	pvS := []runtime.Object{}
	names := []string{}
	nodeLabels := []map[string]string{}
	for i := 0; i < size; i++ {
		index := dcIndex*size + i
		pvS = append(pvS, &v1core.PersistentVolume{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf(nameFormat, dcIndex, i),
				Namespace: r.namespace,
				Labels: map[string]string{
					utils.KubeHostName: fmt.Sprintf(nodeFormat, index),
				},
			},
		})
		names = append(names, fmt.Sprintf(nameFormat, dcIndex, i))
		nodeLabels = append(nodeLabels, map[string]string{utils.KubeHostName: fmt.Sprintf(nodeFormat, index)})
	}

	r.runtimeObjects = append(r.runtimeObjects, pvS...)
	return names, nodeLabels
}

func (r *RuntimeObjectBuilder) GenerateCM(data map[string][]byte) {
	r.runtimeObjects = append(r.runtimeObjects, &v1core.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.namespace,
			Name:      utils.CassandraConfiguration,
		},
		BinaryData: data,
	})
}
