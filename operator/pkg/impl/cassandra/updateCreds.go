package cassandra

import (
	"fmt"

	"github.com/Netcracker/qubership-cassandra-operator/api/v1alpha1"
	"github.com/Netcracker/qubership-cassandra-operator/pkg/impl/utils"
	"github.com/Netcracker/qubership-credential-manager/pkg/manager"
	"github.com/Netcracker/qubership-nosqldb-operator-core/pkg/constants"
	"github.com/Netcracker/qubership-nosqldb-operator-core/pkg/core"
	"go.uber.org/zap"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type UpdateCassandraCredentials struct {
	core.DefaultExecutable
}

func (r *UpdateCassandraCredentials) Execute(ctx core.ExecutionContext) error {
	spec := ctx.Get(constants.ContextSpec).(*v1alpha1.CassandraDeployment)
	cassandraHelperImpl := ctx.Get(utils.CassandraHelperImpl).(utils.CassandraUtils)
	scheme := ctx.Get(constants.ContextSchema).(*runtime.Scheme)
	log := ctx.Get(constants.ContextLogger).(*zap.Logger)

	log.Info("Updating cassandra password")

	err := manager.ActualizeCreds(spec.Spec.Cassandra.SecretName, func(newSecret, oldSecret *v1.Secret) error {

		var secret client.Object = newSecret
		err := controllerutil.SetControllerReference(spec, secret, scheme)
		if err != nil {
			return fmt.Errorf("failed to set owner reference to new secret %v, err: %w", newSecret.Name, err)
		}
		return cassandraHelperImpl.UpdateUserPass(ctx, spec.Spec.User, string(oldSecret.Data[utils.Password]), string(newSecret.Data[utils.Password]))
	})
	return err
}
