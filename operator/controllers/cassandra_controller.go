/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/Netcracker/qubership-cassandra-operator/api/v1alpha1"
	"github.com/Netcracker/qubership-cassandra-operator/pkg/impl"
	"github.com/Netcracker/qubership-cassandra-operator/pkg/impl/cassandra"
	"github.com/Netcracker/qubership-nosqldb-operator-core/pkg/core"
	"github.com/Netcracker/qubership-nosqldb-operator-core/pkg/types"
)

// CassandraReconciler reconciles a Cassandra object
type CassandraReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	Reconciler reconcile.Reconciler
}

//+kubebuilder:rbac:groups=netcracker.com,resources=cassandras,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=netcracker.com,resources=cassandras/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=netcracker.com,resources=cassandras/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *CassandraReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	return r.Reconciler.Reconcile(ctx, req)
}

// SetupWithManager sets up the controller with the Manager.
func (r *CassandraReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Reconciler = newCassandraReconciler(mgr)
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.CassandraDeployment{}).
		Complete(r)
}

func newCassandraReconciler(mgr ctrl.Manager) reconcile.Reconciler {
	return &core.ReconcileCommonService{
		Client:           mgr.GetClient(),
		KubeConfig:       mgr.GetConfig(),
		Scheme:           mgr.GetScheme(),
		Executor:         core.DefaultExecutor(),
		Builder:          &impl.CassandraBuilder{},
		PredeployBuilder: &impl.PreDeployCassandraBuilder{},
		Reconciler:       NewCassandraInstanceReconciler(),
	}
}

// blank assignment to verify that ReconcileCassandraService implements reconcile.Reconciler
var _ reconcile.Reconciler = &core.ReconcileCommonService{}

type CassandraInstanceReconciler struct {
	Instance *v1alpha1.CassandraDeployment
}

func (s *CassandraInstanceReconciler) GetConsulRegistration() *types.ConsulRegistration {
	return nil
}

func (s *CassandraInstanceReconciler) GetConsulServiceRegistrations() map[string]*types.AgentServiceRegistration {
	return nil
}

func NewCassandraInstanceReconciler() core.CommonReconciler {
	return &CassandraInstanceReconciler{}
}

func (s *CassandraInstanceReconciler) GetConfigMapName() string {
	return "cassandra-last-applied-configuration-info"
}

func (s *CassandraInstanceReconciler) SetServiceInstance(client client.Client, request reconcile.Request) {
	cassandraServiceList := &v1alpha1.CassandraDeploymentList{}
	err := core.ListRuntimeObjectsByNamespace(cassandraServiceList, client, request.Namespace)
	if err != nil {
		if errors.IsNotFound(err) {

		}
	}
	msCount := len(cassandraServiceList.Items)
	if msCount != 1 {
		//r.reqLogger.Error("There are " + fmt.Sprintf("%v", msCount) + " instances of Cassandraservice. Please leave only one.")
	}
	s.Instance = &cassandraServiceList.Items[0]
}

func (s *CassandraInstanceReconciler) UpdateStatus(condition types.ServiceStatusCondition) {
	s.Instance.Status.Conditions = []types.ServiceStatusCondition{condition}
}

func (s *CassandraInstanceReconciler) GetStatus() *types.ServiceStatusCondition {
	if len(s.Instance.Status.Conditions) > 0 {
		return &s.Instance.Status.Conditions[0]
	}
	return nil
}

func (s *CassandraInstanceReconciler) GetSpec() interface{} {
	return s.Instance.Spec
}

func (s *CassandraInstanceReconciler) GetInstance() client.Object {
	return s.Instance
}

func (s *CassandraInstanceReconciler) GetDeploymentVersion() string {
	return s.Instance.Spec.DeploymentVersion
}

func (s *CassandraInstanceReconciler) GetVaultRegistration() *types.VaultRegistration {
	return &s.Instance.Spec.VaultRegistration
}

func (s *CassandraInstanceReconciler) UpdateDRStatus(status types.DisasterRecoveryStatus) {

}

func (s *CassandraInstanceReconciler) UpdatePassword() core.Executable {

	return &cassandra.UpdateCassandraCredentials{}
}

func (s *CassandraInstanceReconciler) GetAdminSecretName() string {
	return s.Instance.Spec.SecretName
}

func (s *CassandraInstanceReconciler) UpdatePassWithFullReconcile() bool {
	return false
}

func (s *CassandraInstanceReconciler) GetMessage() string {
	if len(s.Instance.Status.Conditions) > 0 {
		return s.Instance.Status.Conditions[0].Message
	}

	return ""
}
