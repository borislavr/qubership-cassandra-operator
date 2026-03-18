package utils

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/Netcracker/qubership-cassandra-operator/api/v1alpha1"
	"github.com/Netcracker/qubership-cql-driver"
	"github.com/Netcracker/qubership-nosqldb-operator-core/pkg/constants"
	"github.com/Netcracker/qubership-nosqldb-operator-core/pkg/core"
	"github.com/gocql/gocql"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type CassandraUtils interface {
	FindPodsByLabels(ctx core.ExecutionContext, labels map[string]string) (*v1.PodList, error)
	NewClusterBuilder(ctx core.ExecutionContext) cql.ClusterBuilder
	RunSshOnPod(pod *v1.Pod, ctx core.ExecutionContext, cmd string) (string, error)
	GetPodLogs(pod *v1.Pod, ctx core.ExecutionContext, tailLines *int64, previous bool) (string, error)
	GetAllCassandraPods(ctx core.ExecutionContext) (*v1.PodList, error)
	GetConnectionHosts(ctx core.ExecutionContext) []string
	GetReplicationFactor(ctx core.ExecutionContext) string
	CheckLogin(ctx core.ExecutionContext, username string, password string) bool
	IsAllDCsDeployed(ctx core.ExecutionContext) bool
	CreateUser(ctx core.ExecutionContext, username string, password string) error
	UpdateUserPass(ctx core.ExecutionContext, username, oldPassword, newPassword string) error
	ExecuteFlushData(ctx core.ExecutionContext) error
	ExecuteNodetoolCommand(ctx core.ExecutionContext, command string) error
}

var _ CassandraUtils = &CassandraUtilsImpl{}

type CassandraUtilsImpl struct {
	KubernetesHelperImpl core.KubernetesHelper
}

func (r *CassandraUtilsImpl) CreateUser(ctx core.ExecutionContext, username string, password string) error {
	log := ctx.Get(constants.ContextLogger).(*zap.Logger)
	cassandraHelperImpl := ctx.Get(CassandraHelperImpl).(CassandraUtils)

	return cql.ExecInAutoCloseSession(
		cassandraHelperImpl.NewClusterBuilder(ctx).
			WithConsistency(gocql.LocalOne).
			WithUser(Cassandra).
			WithPassword(func() string { return Cassandra }).Build(),
		func(session cql.Session) error {
			log.Debug(fmt.Sprintf("Creating user %s", username))
			if username != Cassandra {
				return session.Query(fmt.Sprintf(
					"CREATE ROLE IF NOT EXISTS '%s' with SUPERUSER = true AND LOGIN = true and PASSWORD = '%s'", username, password)).Exec(false)
			} else {
				return session.Query(fmt.Sprintf("ALTER ROLE '%s' with PASSWORD = '%s'", username, password)).Exec(true)
			}
		})
}

func (r *CassandraUtilsImpl) UpdateUserPass(ctx core.ExecutionContext, username, oldPassword, newPassword string) error {
	log := ctx.Get(constants.ContextLogger).(*zap.Logger)
	cassandraHelperImpl := ctx.Get(CassandraHelperImpl).(CassandraUtils)

	return cql.ExecInAutoCloseSession(
		cassandraHelperImpl.NewClusterBuilder(ctx).
			WithConsistency(gocql.LocalOne).
			WithUser(username).
			WithPassword(func() string { return oldPassword }).Build(),
		func(session cql.Session) error {
			log.Debug(fmt.Sprintf("Updating user %s password", username))
			return session.Query(fmt.Sprintf("ALTER ROLE '%s' with PASSWORD = '%s'", username, newPassword)).Exec(true)

		})
}

func (r *CassandraUtilsImpl) GetReplicationFactor(ctx core.ExecutionContext) string {
	spec := ctx.Get(constants.ContextSpec).(*v1alpha1.CassandraDeployment)

	var dcToReplicasReplication []string

	if r.IsAllDCsDeployed(ctx) {
		for _, dc := range spec.Spec.Cassandra.DeploymentSchema.DataCenters {
			dcToReplicasReplication = append(dcToReplicasReplication, fmt.Sprintf("'%s': '%v'", dc.Name, dc.GetActiveReplicasLen()))
		}
	} else {
		dcs := FilterDC(spec.Spec.Cassandra.DeploymentSchema.DataCenters, func(dc *v1alpha1.DataCenter) bool { return dc.Deploy })
		for _, dc := range dcs {
			dcToReplicasReplication = append(dcToReplicasReplication, fmt.Sprintf("'%s': '%v'", dc.Name, dc.GetActiveReplicasLen()))
		}
	}

	var replication string
	if len(dcToReplicasReplication) > 0 {
		replication = strings.Join(dcToReplicasReplication[:], ",")
	}

	return replication
}
func (r *CassandraUtilsImpl) CheckLogin(ctx core.ExecutionContext, username string, password string) bool {
	cassandraHelperImpl := ctx.Get(CassandraHelperImpl).(CassandraUtils)
	cluster := cassandraHelperImpl.NewClusterBuilder(ctx).WithConsistency(gocql.One).WithUser(username).WithPassword(func() string { return password }).Build()
	sessionError := cql.ExecInAutoCloseSession(cluster, func(session cql.Session) error { return nil })
	return sessionError == nil
}

func (r *CassandraUtilsImpl) GetAllCassandraPods(ctx core.ExecutionContext) (*v1.PodList, error) {
	labels := map[string]string{
		Service: CassandraCluster,
	}

	list, err := r.FindPodsByLabels(ctx, labels)
	if err != nil {
		return nil, err
	}

	if len(list.Items) == 0 {
		return nil, &core.ExecutionError{Msg: "Cassandra pods not found"}
	}

	return list, nil
}
func (r *CassandraUtilsImpl) GetConnectionHosts(ctx core.ExecutionContext) []string {
	spec := ctx.Get(constants.ContextSpec).(*v1alpha1.CassandraDeployment)
	namespace := ctx.Get(constants.ContextRequest).(reconcile.Request).Namespace

	dcs := spec.Spec.Cassandra.DeploymentSchema.DataCenters

	var hosts []string
	for _, dc := range dcs {
		serviceHost := CalcServiceHostName(true, dc, namespace, GetClusterDomain(dc))

		for replica := 0; replica < dc.Replicas; replica++ {
			hosts = append(hosts, fmt.Sprintf("cassandra%d-0.%s", replica, serviceHost))
		}
	}

	return hosts
}

func (r *CassandraUtilsImpl) FindPodsByLabels(ctx core.ExecutionContext, labels map[string]string) (*v1.PodList, error) {
	request := ctx.Get(constants.ContextRequest).(reconcile.Request)

	return r.KubernetesHelperImpl.ListPods(
		request.Namespace,
		labels,
	)
}

func (r *CassandraUtilsImpl) RunSshOnPod(pod *v1.Pod, ctx core.ExecutionContext, cmd string) (string, error) {
	kubeConfig := ctx.Get(constants.ContextKubeClient).(*rest.Config)
	log := ctx.Get(constants.ContextLogger).(*zap.Logger)

	log.Debug(fmt.Sprintf("Command to be executed on %s is: %s", pod.Name, cmd))
	return r.KubernetesHelperImpl.ExecRemote(
		log,
		kubeConfig,
		pod.Name,
		pod.Namespace,
		pod.Spec.Containers[0].Name,
		BashCommand,
		[]string{cmd})
}

func (r *CassandraUtilsImpl) GetPodLogs(pod *v1.Pod, ctx core.ExecutionContext, tailLines *int64, previous bool) (string, error) {
	kubeConfig := ctx.Get(constants.ContextKubeClient).(*rest.Config)
	return r.KubernetesHelperImpl.GetPodLogs(
		kubeConfig,
		pod.Name,
		pod.Namespace,
		pod.Spec.Containers[0].Name,
		tailLines,
		previous)
}

func (r *CassandraUtilsImpl) IsAllDCsDeployed(ctx core.ExecutionContext) bool {
	log := ctx.Get(constants.ContextLogger).(*zap.Logger)
	spec := ctx.Get(constants.ContextSpec).(*v1alpha1.CassandraDeployment)
	cassandraHelperImpl := ctx.Get(CassandraHelperImpl).(CassandraUtils)
	existingDC := NewStream(spec.Spec.Cassandra.DeploymentSchema.DataCenters).FindFirst(func(dc interface{}) bool {
		return !dc.(*v1alpha1.DataCenter).Deploy
	})

	//check if there are multiple kubernetes dc
	if existingDC != nil {
		existingDC = existingDC.(*v1alpha1.DataCenter)
		log.Debug("There is dc with deploy = false, check if it's already deployed")
		session, err := cql.GetSession(cassandraHelperImpl.NewClusterBuilder(ctx).Build(), gocql.LocalOne)
		core.PanicError(err, log.Error, "failed to create cassandra session")
		totalReplicasToDeploy := NewStream(spec.Spec.Cassandra.DeploymentSchema.DataCenters).
			Map(func(d interface{}) interface{} {
				if dc, ok := d.(*v1alpha1.DataCenter); ok {
					return dc.GetActiveReplicasLen()
				}
				core.PanicError(errors.New("type assertion error"), log.Error, "")
				return 0
			}).Sum()

		replicasDeployedIter := session.Query("select count(*) from system.peers").Iter()
		replicasDeployed := -1
		if !replicasDeployedIter.Scan(&replicasDeployed) {
			log.Warn("Could not count peers")
		}
		replicasDeployedIter.Close()

		//check if all replicas deployed
		if totalReplicasToDeploy == replicasDeployed+1 {
			return true
		}
	}
	return false
}

func (r *CassandraUtilsImpl) NewClusterBuilder(ctx core.ExecutionContext) cql.ClusterBuilder {
	spec := ctx.Get(constants.ContextSpec).(*v1alpha1.CassandraDeployment)
	// specServices := ctx.Get(constants.ContextSpec).(*v2.CassandraService)
	// request := ctx.Get(constants.ContextRequest).(reconcile.Request)
	// log := ctx.Get(constants.ContextLogger).(*zap.Logger)
	// client := ctx.Get(constants.ContextClient).(client.Client)
	var hosts []string
	var secret *v1.Secret
	// var err error
	var accessKeyId, secretAccessKey, region string
	dcName := ""
	if spec.Spec.Cassandra.Install {
		hosts = r.GetConnectionHosts(ctx)
		currentDc := NewStream(spec.Spec.Cassandra.DeploymentSchema.DataCenters).FindFirst(func(dc interface{}) bool {
			return dc.(*v1alpha1.DataCenter).Deploy
		}).(*v1alpha1.DataCenter)

		dcName = currentDc.Name
	}
	// else {
	// 	for _, host := range strings.Split(specServices.Spec.AWSKeyspaces.Host, ",") {
	// 		hosts = append(hosts, strings.TrimSpace(host))
	// 	}
	// }
	// if specServices.Spec.AWSKeyspaces.Install {
	// 	secret, err = core.ReadSecret(client, specServices.Spec.AWSKeyspaces.SecretName, request.Namespace)
	// 	core.PanicError(err, log.Error, "Cassandra secret reading failed")
	// }
	if secret != nil {
		if secret.Data[AccessKey] != nil {
			accessKeyId = string(secret.Data[AccessKey])
		} else {
			accessKeyId = ""
		}
		if secret.Data[SecretKey] != nil {
			secretAccessKey = string(secret.Data[SecretKey])
		} else {
			secretAccessKey = ""
		}
		if secret.Data[Region] != nil {
			region = string(secret.Data[Region])
		} else {
			region = ""
		}
	}
	return &cql.ClusterBuilderImpl{
		Host:           hosts,
		User:           spec.Spec.Cassandra.User,
		Password:       func() string { return ctx.Get(ContextPasswordKey).(string) },
		DCName:         dcName,
		Keyspace:       "system",
		Consistency:    gocql.Quorum,
		ConnectTimeout: spec.Spec.GocqlConnectTimeout,
		Timeout:        spec.Spec.GocqlTimeout,
		TlsEnabled:     spec.Spec.TLS.Enabled,
		RootCertPath:   RootCertPath + spec.Spec.TLS.RootCAFileName,
		// AWS:             specServices.Spec.AWSKeyspaces.Install,
		AWS:             false,
		AccessKeyId:     accessKeyId,
		SecretAccessKey: secretAccessKey,
		Region:          region,
	}

}

// todo last two args can be replaced with one - object
func CreateRuntimeObjectContextWrapper(ctx core.ExecutionContext, object client.Object, meta v12.ObjectMeta) error {
	scheme := ctx.Get(constants.ContextSchema).(*runtime.Scheme)
	// spec := ctx.Get(constants.ContextSpec).(*v12.DbaasRedisAdapter)
	helper := ctx.Get(constants.KubernetesHelperImpl).(core.KubernetesHelper)
	// specPointer := &(*spec)

	return helper.CreateRuntimeObject(scheme, nil, object, meta)
}

func CalcReplicaIndex(allDataCenters []*v1alpha1.DataCenter, currentDc int, currentReplicaIndex int) int {
	sum := 0
	separateDC := NewStream(allDataCenters).AnyMatch(func(d interface{}) bool {
		return !d.(*v1alpha1.DataCenter).Deploy
	})
	for dcIndex, dc := range allDataCenters {
		if dcIndex < currentDc && !separateDC {
			sum += dc.Replicas
		} else {
			sum += currentReplicaIndex
			return sum
		}
	}

	return sum
}

func CalcServiceHostName(isCommonService bool, dataCenter *v1alpha1.DataCenter, namespace string, clusterDomain string) string {
	service := Cassandra
	if !isCommonService && dataCenter != nil {
		service = fmt.Sprintf(CassandraDCFormat, dataCenter.Name)
	}

	nameSpacedService := fmt.Sprintf("%s.%s", service, namespace)

	return fmt.Sprintf(ClusterDomainTemplate, nameSpacedService, clusterDomain)
}

func CalcReplicaHostName(isCommonService bool, allDataCenters []*v1alpha1.DataCenter, currentDc int, currentReplicaIndex int, namespace string) string {
	replicaNum := CalcReplicaIndex(allDataCenters, currentDc, currentReplicaIndex)
	replica := fmt.Sprintf(CassandraReplicaNameFormat, replicaNum)
	ss := fmt.Sprintf(StatefulSetPodNameTemplate, replica)

	serviceHost := CalcServiceHostName(isCommonService, allDataCenters[currentDc], namespace, GetClusterDomain(allDataCenters[currentDc]))

	return fmt.Sprintf("%s.%s", ss, serviceHost)
}

func GetClusterDomain(dc *v1alpha1.DataCenter) string {
	if dc.Deploy {
		return DefaultClusterDomain

	} else {
		return dc.ClusterDomain
	}
}

func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func FilterDC(input []*v1alpha1.DataCenter, filter func(dc *v1alpha1.DataCenter) bool) []*v1alpha1.DataCenter {
	var output []*v1alpha1.DataCenter
	for _, dc := range input {
		if filter(dc) {
			output = append(output, dc)
		}
	}

	return output
}

func (r *CassandraUtilsImpl) ExecuteFlushData(ctx core.ExecutionContext) error {
	return r.ExecuteNodetoolCommand(ctx, "nodetool flush")
}

func (r *CassandraUtilsImpl) ExecuteNodetoolCommand(ctx core.ExecutionContext, command string) error {
	cassandraHelperImpl := ctx.Get(CassandraHelperImpl).(CassandraUtils)
	command = fmt.Sprintf("%s 2> /dev/null", command)

	log := ctx.Get(constants.ContextLogger).(*zap.Logger)
	log.Info(fmt.Sprintf("Cassandra %s step is started", command))

	list, err := cassandraHelperImpl.GetAllCassandraPods(ctx)
	core.PanicError(err, log.Error, "Cassandra pods listing failed")

	var wg sync.WaitGroup

	for _, pod := range list.Items {
		wg.Add(1)

		go func(pod v1.Pod) {
			defer wg.Done()

			log.Info(fmt.Sprintf("Executing %s on %s", command, pod.ObjectMeta.Name))
			_, err := cassandraHelperImpl.RunSshOnPod(&pod, ctx, command)
			if err != nil {
				log.Warn(fmt.Sprintf("Could not perform command %s on %s. Error: %s"+
					"If necessary perform %s manually", command, pod.ObjectMeta.Name, err, command))
			} else {
				log.Info(fmt.Sprintf("%s was successfully executed on the %s", command, pod.ObjectMeta.Name))
			}
		}(pod)
	}

	wg.Wait()

	return nil
}
