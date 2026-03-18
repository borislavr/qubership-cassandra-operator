package common

import (
	"github.com/Netcracker/qubership-cassandra-operator/api/v1alpha1"
	"github.com/Netcracker/qubership-cassandra-operator/pkg/impl/utils"
	"github.com/Netcracker/qubership-nosqldb-operator-core/pkg/constants"
	"github.com/Netcracker/qubership-nosqldb-operator-core/pkg/core"
	"go.uber.org/zap"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type SetPasswordFromSecret struct {
	core.DefaultExecutable
}

func (r *SetPasswordFromSecret) Execute(ctx core.ExecutionContext) error {
	request := ctx.Get(constants.ContextRequest).(reconcile.Request)
	spec := ctx.Get(constants.ContextSpec).(*v1alpha1.CassandraDeployment)
	log := ctx.Get(constants.ContextLogger).(*zap.Logger)
	client := ctx.Get(constants.ContextClient).(client.Client)

	log.Debug("Cassandra set password from secret is started")

	secret, err := core.ReadSecret(client, spec.Spec.Cassandra.SecretName, request.Namespace)
	core.PanicError(err, log.Error, "Cassandra secret reading failed")
	ctx.Set(utils.ContextPasswordKey, string(secret.Data[utils.Password]))
	log.Debug("Cassandra set password from secret is ended")

	return nil
}
