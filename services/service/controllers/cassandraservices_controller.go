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
	"fmt"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8type "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/Netcracker/qubership-cassandra-supplementary/api/v1alpha1"
	impl "github.com/Netcracker/qubership-cassandra-supplementary/pkg"
	"github.com/Netcracker/qubership-nosqldb-operator-core/pkg/core"
	"github.com/Netcracker/qubership-nosqldb-operator-core/pkg/types"
)

var setupLog = ctrl.Log.WithName("setup")

// CassandraSupplServiceReconciler reconciles a CassandraService object
type CassandraSupplServiceReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	Reconciler reconcile.Reconciler
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *CassandraSupplServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	err := WaitForCassandraOperatorReady(r.Client, "cassandra-operator", req.Namespace)
	if err != nil {
		setupLog.Info("Cassandra Operator is not ready...")
		return ctrl.Result{RequeueAfter: 15 * time.Second}, nil
	}

	_ = log.FromContext(ctx)
	return r.Reconciler.Reconcile(ctx, req)
}

func WaitForCassandraOperatorReady(k8sClient client.Client, name, namespace string) error {
	setupLog.Info("Waiting for Cassandra CR to be ready...")
	ctx := context.Background()

	cassandraGVK := schema.GroupVersionKind{
		Group:   "netcracker.com",
		Version: "v1alpha1",
		Kind:    "CassandraDeployment",
	}

	return wait.PollUntilContextTimeout(ctx, 5*time.Second, 10*time.Minute, true, func(ctx context.Context) (bool, error) {
		cassandraCR := &unstructured.Unstructured{}
		cassandraCR.SetGroupVersionKind(cassandraGVK)

		err := k8sClient.Get(ctx, k8type.NamespacedName{
			Name:      name,
			Namespace: namespace,
		}, cassandraCR)
		if err != nil {
			return false, err
		}

		status, found, err := unstructured.NestedSlice(cassandraCR.Object, "status", "conditions")
		if !found || err != nil {
			return false, fmt.Errorf("unable to find status.conditions in CassandraCluster")
		}

		for _, cond := range status {
			condMap, ok := cond.(map[string]interface{})
			if !ok {
				continue
			}

			t, _ := condMap["type"].(string)
			s, ok := condMap["status"].(bool) // read as bool instead of string

			if !ok {
				continue
			}

			switch strings.ToLower(t) {
			case "successful":
				if s {
					setupLog.Info("Cassandra CR status is Successful")
					return true, nil
				}
			case "failed":
				if s {
					setupLog.Error(nil, "Cassabdra CR failed")
					return true, fmt.Errorf("Cassabdra CR failed")
				}
			}
			setupLog.Info("Waiting for Cassabdra CR to be ready", "type", t, "status", s)
		}

		return false, nil
	})
}

// SetupWithManager sets up the controller with the Manager.
func (r *CassandraSupplServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Reconciler = newCassandraServiceReconciler(mgr)
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.CassandraSupplService{}).
		Complete(r)
}

func newCassandraServiceReconciler(mgr ctrl.Manager) reconcile.Reconciler {
	return &core.ReconcileCommonService{
		Client:           mgr.GetClient(),
		KubeConfig:       mgr.GetConfig(),
		Scheme:           mgr.GetScheme(),
		Executor:         core.DefaultExecutor(),
		Builder:          &impl.CassandraServiceBuilder{},
		PredeployBuilder: &impl.PreDeployBuilder{},
		Reconciler:       NewCassandraServiceInstanceReconciler(),
	}
}

// blank assignment to verify that ReconcileCassandraService implements reconcile.Reconciler
var _ reconcile.Reconciler = &core.ReconcileCommonService{}

type CassandraServiceInstanceReconciler struct {
	Instance *v1alpha1.CassandraSupplService
}

func NewCassandraServiceInstanceReconciler() core.CommonReconciler {
	return &CassandraServiceInstanceReconciler{}
}

func (s *CassandraServiceInstanceReconciler) GetConfigMapName() string {
	return "cassandra-services-last-applied-configuration-info"
}

func (s *CassandraServiceInstanceReconciler) GetConsulRegistration() *types.ConsulRegistration {
	return &s.Instance.Spec.ConsulRegistration
}

func (s *CassandraServiceInstanceReconciler) GetConsulServiceRegistrations() map[string]*types.AgentServiceRegistration {
	return s.Instance.Spec.ConsulDiscoverySettings
}

func (s *CassandraServiceInstanceReconciler) SetServiceInstance(client client.Client, request reconcile.Request) {
	cassandraServiceList := &v1alpha1.CassandraSupplServiceList{}
	err := core.ListRuntimeObjectsByNamespace(cassandraServiceList, client, request.Namespace)
	if err != nil {
		msCount := len(cassandraServiceList.Items)
		if errors.IsNotFound(err) || msCount == 0 {
			panic(fmt.Sprintf("No service instance found, err: %v", err))
		}
	}

	s.Instance = &cassandraServiceList.Items[0]
}

func (s *CassandraServiceInstanceReconciler) UpdateStatus(condition types.ServiceStatusCondition) {
	s.Instance.Status.Conditions = []types.ServiceStatusCondition{condition}
}

func (s *CassandraServiceInstanceReconciler) GetStatus() *types.ServiceStatusCondition {
	if len(s.Instance.Status.Conditions) > 0 {
		return &s.Instance.Status.Conditions[0]
	}
	return nil
}

func (s *CassandraServiceInstanceReconciler) GetSpec() interface{} {
	return s.Instance.Spec
}

func (s *CassandraServiceInstanceReconciler) GetInstance() client.Object {
	return s.Instance
}

func (s *CassandraServiceInstanceReconciler) GetDeploymentVersion() string {
	return s.Instance.Spec.DeploymentVersion
}

func (s *CassandraServiceInstanceReconciler) GetVaultRegistration() *types.VaultRegistration {
	return &s.Instance.Spec.VaultRegistration
}

func (s *CassandraServiceInstanceReconciler) UpdateDRStatus(status types.DisasterRecoveryStatus) {

}

func (s *CassandraServiceInstanceReconciler) UpdatePassword() core.Executable {
	return nil
}

func (s *CassandraServiceInstanceReconciler) UpdatePassWithFullReconcile() bool {
	return true
}

func (s *CassandraServiceInstanceReconciler) GetAdminSecretName() string {
	return s.Instance.Spec.Cassandra.SecretName
}

func (s *CassandraServiceInstanceReconciler) GetMessage() string {
	if len(s.Instance.Status.Conditions) > 0 {
		return s.Instance.Status.Conditions[0].Message
	}

	return ""
}
