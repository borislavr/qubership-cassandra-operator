package backup

import (
	"fmt"
	"strings"

	"github.com/Netcracker/qubership-cql-driver"
	"github.com/Netcracker/qubership-cassandra-supplementary/api/v1alpha1"
	"github.com/Netcracker/qubership-cassandra-supplementary/pkg/utils"

	"github.com/Netcracker/qubership-nosqldb-operator-core/pkg/constants"
	"github.com/Netcracker/qubership-nosqldb-operator-core/pkg/core"
	"github.com/Netcracker/qubership-nosqldb-operator-core/pkg/vault"
	"github.com/gocql/gocql"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type BackupSSHKeyStep struct {
	core.DefaultExecutable
}

func (r *BackupSSHKeyStep) Execute(ctx core.ExecutionContext) error {
	request := ctx.Get(constants.ContextRequest).(reconcile.Request)
	spec := ctx.Get(constants.ContextSpec).(*v1alpha1.CassandraSupplService)
	client := ctx.Get(constants.ContextClient).(client.Client)
	helperImpl := ctx.Get(utils.KubernetesHelperImpl).(core.KubernetesHelper)
	kubeConfig := ctx.Get(constants.ContextKubeClient).(*rest.Config)

	clusterBuilder := ctx.Get(utils.ContextClusterBuilder).(cql.ClusterBuilder)

	tries := core.MaxInt(ctx.Get(utils.TriesCount).(int), 1)
	log := ctx.Get(constants.ContextLogger).(*zap.Logger)

	secret, secErr := core.ReadSecret(client, spec.Spec.Cassandra.SecretName, request.Namespace)
	core.PanicError(secErr, log.Error, fmt.Sprintf("Failed to read secret %s", spec.Spec.Cassandra.SecretName))

	log.Info("SSH Key Step for Backup started")

	var dcToReplicasReplication []string

	for _, dc := range spec.Spec.Cassandra.DeploymentSchema.DataCenters {
		dcToReplicasReplication = append(dcToReplicasReplication, fmt.Sprintf("'%s': '%v'", dc.Name, dc.Replicas))
	}

	if len(dcToReplicasReplication) == 0 {
		core.PanicError(fmt.Errorf("failed to calculate replication parameters"), log.Error, "Failed to calculate replication parameters")
	}
	var replication string = strings.Join(dcToReplicasReplication[:], ",")

	pass := string(secret.Data[utils.Password])
	var vaultErr error
	if spec.Spec.VaultRegistration.Enabled {
		vaultHelper := ctx.Get(constants.ContextVault).(vault.VaultHelper)
		pass, vaultErr = vaultHelper.ResolvePassword(pass)
		if vaultErr != nil {
			return vaultErr
		}
	}

	cluster := clusterBuilder.WithHost(core.OptionalString(spec.Spec.Cassandra.Host, fmt.Sprintf("%s.%s", utils.Cassandra, request.Namespace))).
		WithUser(string(secret.Data[utils.Username])).
		WithPassword(func() string { return pass }).
		WithRootCertPath(utils.RootCertPath + spec.Spec.TLS.RootCAFileName).
		WithTLSEnabled(spec.Spec.TLS.Enabled).
		WithKeyspace("system").
		WithConsistency(gocql.Quorum).Build()

	session, sessionErr := cql.GetSession(cluster, gocql.Quorum)
	core.PanicError(sessionErr, log.Error, "failed to create cassandra session")
	var publicId string
	var privateId string
	var publicKey string
	var privateKey string
	var publicIdRsa string
	var privateIdRsa string
	var err error
	publicKeyIterator := session.Query("SELECT id, key FROM ssh.backup WHERE id = ?", "public").Iter()
	privateKeyIterator := session.Query("SELECT id, key FROM ssh.backup WHERE id = ?", "private").Iter()
	// if there is no public or private key
	if !publicKeyIterator.Scan(&publicId, &publicKey) || !privateKeyIterator.Scan(&privateId, &privateKey) {
		log.Info("No ssh keys found in database, generating new ones")

		publicIdRsa, privateIdRsa, err = utils.GenerateKeyPair()
		core.PanicError(err, log.Error, "SHH keys not generated")

		// session.SetConsistency(gocql.LocalOne)
		session.Query(fmt.Sprintf("CREATE KEYSPACE if not exists ssh WITH REPLICATION = {'class' : 'NetworkTopologyStrategy', %s }; ", replication)).Exec(true)
		session.Query("CREATE TABLE if not exists ssh.backup ( id text PRIMARY KEY, key text)").Exec(true)
		session.Query("INSERT INTO ssh.backup (id, key)  VALUES (?, ?)", "public", publicIdRsa).Exec(true)
		session.Query("INSERT INTO ssh.backup (id, key)  VALUES (?, ?)", "private", privateIdRsa).Exec(true)

	} else {
		log.Info("ssh keys found in database, setting keys to pods")

		session.Query(fmt.Sprintf("alter KEYSPACE ssh  WITH REPLICATION = {'class' : 'NetworkTopologyStrategy', %s }; ", replication)).Exec(true)

		publicIdRsa = publicKey
		privateIdRsa = privateKey
	}

	sshSecret := &corev1.Secret{
		ObjectMeta: v12.ObjectMeta{
			Namespace: request.Namespace,
			Name:      utils.SSHSecret,
		},
		StringData: map[string]string{
			"publicKey":  publicIdRsa,
			"privateKey": privateIdRsa,
		},
	}

	err = utils.CreateRuntimeObjectContextWrapper(ctx, sshSecret, sshSecret.ObjectMeta, utils.BasicLabels{})
	core.PanicError(err, log.Error, "SSH ConfigMap creation failed")

	if err := publicKeyIterator.Close(); err != nil {
		log.Warn(fmt.Sprintf("Failed to close publicKeyIterator: %s", err))
	}
	if err := privateKeyIterator.Close(); err != nil {
		log.Warn(fmt.Sprintf("Failed to close privateKeyIterator: %s", err))
	}

	cassandraLabels := map[string]string{
		utils.Service: utils.CassandraCluster,
	}

	cassandraPodList, err := helperImpl.ListPods(request.Namespace, cassandraLabels)

	if cassandraPodList != nil {
		commands := []struct {
			Command       string
			IgnoreOnError bool
		}{
			{"mkdir -p /var/lib/cassandra/data/.ssh/", false},
			{"rm -rf /var/lib/cassandra/data/.ssh/authorized_keys", false},
			{fmt.Sprintf("echo '%s' > /var/lib/cassandra/data/.ssh/authorized_keys", publicIdRsa), false},
			{"chmod -R 700 /var/lib/cassandra/data/.ssh", false},
			{"chmod 600 /var/lib/cassandra/data/.ssh/authorized_keys", false},
		}

		for _, pod := range cassandraPodList.Items {
			doneTriesCount := 0

			for doneTriesCount < tries {
				var commandError error
				var errMsg string
				for _, command := range commands {
					// commandError
					_, commandError = helperImpl.ExecRemote(log, kubeConfig, pod.Name, pod.Namespace, pod.Spec.Containers[0].Name,
						"bash", []string{command.Command})
					if commandError != nil {
						if command.IgnoreOnError {
							log.Warn(fmt.Sprintf("Command '%s' is failed on '%s' pod with the following message: %s. Will be ignored",
								command.Command, pod.Name, commandError.Error()))

						} else {
							errMsg = fmt.Sprintf("Command '%v+' is failed on '%s' pod", command, pod.Name)
							doneTriesCount++
							break
						}
					}
				}

				if commandError != nil {
					if doneTriesCount == tries {
						core.PanicError(commandError, log.Error, errMsg)
					} else {
						log.Warn(errMsg)
						log.Debug("Trying again...")
					}
				} else {
					break
				}
			}

			log.Debug(fmt.Sprintf("Backup auth keys propagated to '%s'", pod.Name))
		}

	}

	log.Info("SSH Key Step for Backup finished")
	return nil
}
