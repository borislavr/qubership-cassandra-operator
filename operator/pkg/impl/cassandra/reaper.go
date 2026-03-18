package cassandra

import (
	"fmt"

	"github.com/Netcracker/qubership-cassandra-operator/api/v1alpha1"
	"github.com/Netcracker/qubership-cassandra-operator/pkg/impl/utils"
	"github.com/Netcracker/qubership-cql-driver"
	"github.com/Netcracker/qubership-nosqldb-operator-core/pkg/constants"
	"github.com/Netcracker/qubership-nosqldb-operator-core/pkg/core"
	"github.com/gocql/gocql"
	"go.uber.org/zap"
	v1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type CassandraReaper struct {
	core.DefaultExecutable
}

func (r *CassandraReaper) Execute(ctx core.ExecutionContext) error {
	spec := ctx.Get(constants.ContextSpec).(*v1alpha1.CassandraDeployment)
	log := ctx.Get(constants.ContextLogger).(*zap.Logger)
	request := ctx.Get(constants.ContextRequest).(reconcile.Request)
	cassandraHelperImpl := ctx.Get(utils.CassandraHelperImpl).(utils.CassandraUtils)

	session, sessionErr := cql.GetSession(cassandraHelperImpl.NewClusterBuilder(ctx).Build(), gocql.LocalOne)
	core.PanicError(sessionErr, log.Error, "failed to create cassandra session")
	replication := cassandraHelperImpl.GetReplicationFactor(ctx)

	log.Debug(fmt.Sprintf("Reaper DB will be updated with replication %s", replication))

	// create if not created
	session.Query(fmt.Sprintf("CREATE KEYSPACE if not exists reaper_db  WITH REPLICATION = { 'class' : 'NetworkTopologyStrategy',  %s }", replication)).Exec(true)
	// update replication
	session.Query(fmt.Sprintf("alter KEYSPACE reaper_db  WITH REPLICATION = { 'class' : 'NetworkTopologyStrategy',  %s }", replication)).Exec(true)

	var annotations = map[string]string{
		"nginx.ingress.kubernetes.io/affinity":               "cookie",
		"nginx.ingress.kubernetes.io/session-cookie-name":    "route",
		"nginx.ingress.kubernetes.io/session-cookie-expires": "172800",
		"nginx.ingress.kubernetes.io/session-cookie-max-age": "172800",
	}

	if spec.Spec.TLS.Enabled {
		annotations["nginx.ingress.kubernetes.io/backend-protocol"] = "HTTPS"
		annotations["nginx.ingress.kubernetes.io/proxy-ssl-verify"] = "on"
		annotations["nginx.ingress.kubernetes.io/proxy-ssl-name"] = "cassandra." + request.Namespace
		annotations["nginx.ingress.kubernetes.io/proxy-ssl-secret"] = request.Namespace + "/" + spec.Spec.TLS.RootCASecretName
	}

	if spec.Spec.Reaper.IngressHost != "" {
		var pathType v1.PathType = "Prefix"
		ingress := &v1.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "cassandra-reaper",
				Namespace:   request.Namespace,
				Annotations: annotations,
			},
			Spec: v1.IngressSpec{
				Rules: []v1.IngressRule{
					{
						Host: spec.Spec.Reaper.IngressHost,
						IngressRuleValue: v1.IngressRuleValue{
							HTTP: &v1.HTTPIngressRuleValue{
								Paths: []v1.HTTPIngressPath{
									{
										Path:     "/",
										PathType: &pathType,
										Backend: v1.IngressBackend{
											Service: &v1.IngressServiceBackend{
												Name: utils.Cassandra,
												Port: v1.ServiceBackendPort{
													Number: int32(spec.Spec.Reaper.Port),
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}

		err := utils.CreateRuntimeObjectContextWrapper(ctx, ingress, ingress.ObjectMeta)
		core.PanicError(err, log.Error, "Ingress "+ingress.Name+" creation failed")

		log.Debug("Cassandra reaper Ingress have been created")
	}

	return nil
}

func (r *CassandraReaper) Condition(ctx core.ExecutionContext) (bool, error) {
	spec := ctx.Get(constants.ContextSpec).(*v1alpha1.CassandraDeployment)

	return spec.Spec.Reaper.Install, nil
}
