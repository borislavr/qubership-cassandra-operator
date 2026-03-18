package cassandra

import (
	"context"
	"time"

	"github.com/Netcracker/qubership-cassandra-operator/api/v1alpha1"
	"github.com/Netcracker/qubership-cassandra-operator/pkg/impl/utils"
	"github.com/Netcracker/qubership-nosqldb-operator-core/pkg/constants"
	"github.com/Netcracker/qubership-nosqldb-operator-core/pkg/core"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
	v1 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type CassandraConfigurationUpgradeStep struct {
	core.DefaultExecutable
}

func (r *CassandraConfigurationUpgradeStep) Execute(ctx core.ExecutionContext) error {
	kubeClient := ctx.Get(constants.ContextClient).(client.Client)
	request := ctx.Get(constants.ContextRequest).(reconcile.Request)
	log := ctx.Get(constants.ContextLogger).(*zap.Logger)

	cassandraSpec := ctx.Get(constants.ContextSpec).(*v1alpha1.CassandraDeployment)
	configFromSpec := cassandraSpec.Spec.Configuration

	if configFromSpec == "" {
		log.Info("Cassandra ConfigMap: no new config in spec. Updating cluster_name")
	}

	configMapFromCloud := &v1.ConfigMap{}
	err := wait.PollImmediate(time.Second, time.Second*time.Duration(cassandraSpec.Spec.WaitTimeout), func() (done bool, err error) {
		getErr := kubeClient.Get(context.TODO(), types.NamespacedName{Name: utils.CassandraConfiguration, Namespace: request.Namespace}, configMapFromCloud)
		if getErr != nil {
			log.Warn("Could not get cassandra-configuration Config Map")
		}
		return getErr == nil, nil
	})

	core.PanicError(err, log.Error, "Cassandra configuration listing failed")

	data, updated := parseConfiguration(ctx, []byte(configFromSpec), configMapFromCloud)
	if updated {
		cfg := &v1.ConfigMap{
			ObjectMeta: v12.ObjectMeta{
				Namespace: request.Namespace,
				Name:      utils.CassandraConfiguration,
				Labels: map[string]string{
					utils.AppPartOf: cassandraSpec.Spec.PartOf,
				},
			},
			Data: map[string]string{
				utils.Config: string(data),
			},
		}
		err = utils.CreateRuntimeObjectContextWrapper(ctx, cfg, cfg.ObjectMeta)
		core.PanicError(err, log.Error, "Cassandra config changing failed")

		log.Info("Cassandra config has been updated")
	}

	return nil
}

func parseConfiguration(ctx core.ExecutionContext, bytesConfigFromSpec []byte, configMap *v1.ConfigMap) ([]byte, bool) {
	log := ctx.Get(constants.ContextLogger).(*zap.Logger)
	updated := false
	configBytesIn := configMap.Data[utils.Config]
	configFromCloud := make(map[interface{}]interface{})
	err := yaml.Unmarshal([]byte(configBytesIn), &configFromCloud)
	core.HandleError(err, log.Error, "Could not unmarshal passed yaml config!")

	configFromSpec := make(map[interface{}]interface{})
	err = yaml.Unmarshal(bytesConfigFromSpec, &configFromSpec)
	core.HandleError(err, log.Error, "Could not unmarshal passed yaml config!")

	for keyFromSpec, valueFromSpec := range configFromSpec {
		updated = true
		if keyFromSpec == "cluster_name" {
			log.Debug("Setting cluster_name from CR spec")
		}
		configFromCloud[keyFromSpec] = valueFromSpec
	}

	configBytesOut, err := yaml.Marshal(configFromCloud)
	core.HandleError(err, log.Error, "Could not marshal config yaml")
	return configBytesOut, updated
}
