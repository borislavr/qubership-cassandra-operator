package common

import (
	mFiber "github.com/Netcracker/qubership-cassandra-operator/pkg/impl/fiber"
	"github.com/Netcracker/qubership-nosqldb-operator-core/pkg/constants"
	"github.com/Netcracker/qubership-nosqldb-operator-core/pkg/core"
	"go.uber.org/zap"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type RunFiberServer struct {
	core.DefaultExecutable
}

func (r *RunFiberServer) Execute(ctx core.ExecutionContext) error {
	log := ctx.Get(constants.ContextLogger).(*zap.Logger).Named("RunFiberServer")
	namespace := ctx.Get(constants.ContextRequest).(reconcile.Request).Namespace
	service := mFiber.NewCassandraFiberService(ctx)

	return mFiber.RunFiberServer(8069, service, namespace, log)
}
