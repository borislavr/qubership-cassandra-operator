package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/Netcracker/qubership-cassandra-dbaas-adapter/go/impl"
	"github.com/Netcracker/qubership-dbaas-adapter-core/pkg/dao"
	"github.com/Netcracker/qubership-dbaas-adapter-core/pkg/dbaas"
	fiber2 "github.com/Netcracker/qubership-dbaas-adapter-core/pkg/impl/fiber"
	"github.com/Netcracker/qubership-dbaas-adapter-core/pkg/service"
	"github.com/Netcracker/qubership-dbaas-adapter-core/pkg/utils"
	"github.com/gocql/gocql"
	"github.com/gofiber/fiber/v2"

	mUtils "github.com/Netcracker/qubership-cassandra-dbaas-adapter/go/utils"
)

func main() {
	logger := utils.GetLogger(mUtils.GetEnvBool("LOG_DEBUG", false))
	ctxLogger := utils.AddLoggerContext(logger, context.Background())

	appName := "cassandra"

	// Defaults
	appPath := "/" + appName
	profiler := utils.GetEnvAsBool("PROFILER", false)
	namespace := utils.GetEnv("NAMESPACE", "")
	port := utils.GetEnvAsInt("PORT", mUtils.DefaultPort)
	apiUser := mUtils.GetSecret("/var/run/secrets/dbaas-adapter/username", "dbaas-adapter")
	apiPass := mUtils.GetSecret("/var/run/secrets/dbaas-adapter/password", "dbaas-adapter")
	apiVersion := utils.GetEnv("API_VERSION", "v1")
	multiUserEnabled := utils.GetEnvAsBool("MULTI_USERS_ENABLED", false)
	aggregatorAdapterAddress := utils.GetEnv("DBAAS_ADAPTER_ADDRESS", fmt.Sprintf("http://dbaas-%s-adapter.%s:8080", appName, namespace))
	aggregatorRegistrationAddress := utils.GetEnv("DBAAS_AGGREGATOR_REGISTRATION_ADDRESS", "http://dbaas-aggregator.dbaas:8080")
	aggregatorRegistrationIdentifier := utils.GetEnv("DBAAS_AGGREGATOR_PHYSICAL_DATABASE_IDENTIFIER", appName)
	aggregatorRegistrationUser := mUtils.GetSecret("/var/run/secrets/dbaas-aggregator/username", "cluster-dba")
	aggregatorRegistrationPass := mUtils.GetSecret("/var/run/secrets/dbaas-aggregator/password", "Bnmq5567_PO")
	aggregatorRegistrationDelay := utils.GetEnvAsInt("DBAAS_AGGREGATOR_REGISTRATION_FIXED_DELAY_MS", 150000)
	aggregatorRegistrationRetryTime := utils.GetEnvAsInt("DBAAS_AGGREGATOR_REGISTRATION_RETRY_TIME_MS", 60000)
	aggregatorRegistrationRetryDelay := utils.GetEnvAsInt("DBAAS_AGGREGATOR_REGISTRATION_RETRY_DELAY_MS", 5000)

	streamingRoleName := utils.GetEnv("DBAAS_STREAMING_ROLE_NAME", "streaming")
	streamingRolePermissions := mUtils.GetEvAsStringSlice("DBAAS_STREAMING_ROLE_PERMISSIONS", []string{"ALL"})

	// DB Administration
	cassandraHost := utils.GetEnv("CASSANDRA_HOSTNAME", fmt.Sprintf("cassandra.%s", namespace))
	cassandraPort := utils.GetEnvAsInt("CASSANDRA_PORT", 9042)
	cassandraUser := mUtils.GetSecret("/var/run/secrets/cassandra/username", "cassandra")
	cassandraPass := mUtils.GetSecret("/var/run/secrets/cassandra/password", "cassandra")
	defaultTopology := utils.GetEnv("CASSANDRA_DEFAULT_TOPOLOGY", "{'class':'SimpleStrategy','replication_factor': 1 }")
	gocqlConnectTimeout := utils.GetEnvAsInt("GOCQL_CONNECT_TIMEOUT", 20)
	consistecyLevel := utils.GetEnv("GOCQL_CONSISTENCY", "QUORUM")
	defaultKeyspace := utils.GetEnv("GOCQL_DEFAULT_KEYSPACE", "system")
	gocqlTimeout := utils.GetEnvAsInt("GOCQL_TIMEOUT", 20)

	// Backup Daemon Administration
	backupAddress := utils.GetEnv("BACKUP_DAEMON_ADDRESS", fmt.Sprintf("http://%s-backup-daemon:8080", appName))
	backupDaemonApiUser := mUtils.GetSecret("/var/run/secrets/backup/username", "")
	backupDaemonApiUPass := mUtils.GetSecret("/var/run/secrets/backup/password", "")

	var backupAdminServiceImpl service.BackupAdministrationService
	if backupAddress == "" || backupDaemonApiUser == "" || backupDaemonApiUPass == "" {
		ctxLogger.Debug("Backup address or credentials are not set")
	} else {
		client := &http.Client{}
		if strings.Contains(backupAddress, "https") {
			cert := mUtils.GetCACert()
			if cert == "" {
				utils.PanicError(errors.New(""), ctxLogger.Error, "CA Certificate is empty or not set")
			}
			if err := utils.ConfigureHttpsForClientWithCertificate(client, cert); err != nil {
				utils.PanicError(err, ctxLogger.Error, "Failed to set up https client")
			}
		}
		backupAdminServiceImpl = service.DefaultBackupAdministrationService(
			logger,
			backupAddress,
			backupDaemonApiUser,
			backupDaemonApiUPass,
			false,
			client,
			48, []string{"-"})
	}

	// Vault
	isVaultEnabled := utils.GetEnvAsBool("VAULT_ENABLED", false)
	var vaultClient *utils.VaultClient
	if isVaultEnabled {
		vaultConfig := utils.VaultConfig{
			IsVaultEnabled:  isVaultEnabled,
			Address:         utils.GetEnv("VAULT_ADDR", ""),
			VaultRole:       utils.GetEnv("VAULT_ROLE", "cassandra-sa"),
			VaultRotPeriod:  utils.GetEnv("VAULT_ROTATION_PERIOD", "86400"),
			VaultAuthMethod: utils.GetEnv("VAULT_AUTH_METHOD", ""),
			VaultDBName:     utils.GetEnv("VAULT_DB_ENGINE_NAME", "cassandra"),
		}
		vaultClient = utils.NewVaultClient(vaultConfig)
	}

	// Supports
	supports := dao.SupportsBase{
		Users:             true,
		Settings:          true,
		DescribeDatabases: false,
		AdditionalKeys: dao.Supports{
			"backupRestore": backupAdminServiceImpl != nil,
			"vault":         isVaultEnabled,
		},
	}

	basicRegistrationAuth := dao.BasicAuth{
		Username: aggregatorRegistrationUser,
		Password: aggregatorRegistrationPass,
	}

	dbaasClient, err := dbaas.NewDbaasClient(aggregatorRegistrationAddress, &basicRegistrationAuth, nil)
	if dbaasClient == nil {
		panic(fmt.Errorf("Failed to establish connection to DBaaS aggregator, err: %v", err))
	}

	if err != nil {
		ctxLogger.Error(fmt.Sprintf("Failed to get DBaaS aggregator version, err %v. Setting default API version", err))
	}

	version, _ := dbaasClient.GetVersion() // if err != nil it will fail in the condition above
	if version == "v3" {
		apiVersion = "v2"
	} else {
		apiVersion = "v1"
	}

	ctxLogger.Info(fmt.Sprintf("API version is %s", apiVersion))

	var dbAdminImpl service.DbAdministration = impl.GetCassandraDbAdministration(
		logger,
		[]string{"admin", "ro", "rw", streamingRoleName},
		map[string]bool{
			mUtils.FeatureMultiUsers: multiUserEnabled,
			mUtils.FeatureTLS:        utils.IsHttpsEnabled(),
		},
		apiVersion,
		cassandraHost, cassandraPort, cassandraUser, cassandraPass,
		defaultKeyspace, gocql.ParseConsistency(consistecyLevel),
		gocqlConnectTimeout, gocqlTimeout, defaultTopology,
		streamingRoleName, streamingRolePermissions,
	)

	admService := service.NewCoreAdministrationService(
		namespace,
		port,
		dbAdminImpl,
		logger,
		isVaultEnabled,
		vaultClient,
		"",
	)

	log.Fatal(fiber2.RunFiberServer(port, func(app *fiber.App, ctx context.Context) error {
		fiber2.BuildFiberDBaaSAdapterHandlers(
			app,
			apiUser,
			apiPass,
			appPath,
			admService,
			service.NewPhysicalRegistrationService(
				appName,
				ctxLogger,
				aggregatorRegistrationIdentifier,
				aggregatorAdapterAddress,
				dao.BasicAuth{
					Username: apiUser,
					Password: apiPass,
				},
				mUtils.ReadLabelsFile(), //labels
				dbaasClient,
				aggregatorRegistrationDelay,
				aggregatorRegistrationRetryTime,
				aggregatorRegistrationRetryDelay,
				admService,
				ctx,
			),
			backupAdminServiceImpl,
			supports.ToMap(),
			ctxLogger,
			profiler, "")

		prefix := fmt.Sprintf("/api/%s/dbaas/adapter/cassandra/databases/", apiVersion)
		app.Put(prefix+":dbName/settings", dbAdminImpl.(*impl.CassandraDbAdministration).UpdateCassandraSettingsHandler)
		app.Get("/api/version", func(c *fiber.Ctx) error {
			return c.SendString(apiVersion)
		})

		return nil
	}))
}
