package impl

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/Netcracker/qubership-cassandra-dbaas-adapter/go/impl/cassandra"
	mUtils "github.com/Netcracker/qubership-cassandra-dbaas-adapter/go/utils"
	"github.com/Netcracker/qubership-dbaas-adapter-core/pkg/dao"
	"github.com/Netcracker/qubership-dbaas-adapter-core/pkg/service"
	"github.com/Netcracker/qubership-dbaas-adapter-core/pkg/utils"
	"github.com/gocql/gocql"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

var (
	dbNameRegexpExpression   = "^[_A-z0-9]*$"
	dbNameRegexp, _          = regexp.Compile(dbNameRegexpExpression)
	prefixRegexpExpression   = "^[_A-z0-9]*$"
	prefixRegexp, _          = regexp.Compile(prefixRegexpExpression)
	userNameRegexpExpression = "^[_A-z0-9]*$"
	userNameRegexp, _        = regexp.Compile(userNameRegexpExpression)

	replicationKey = "replication"

	apiResourcesToCassandraEntities = map[string]string{
		mUtils.UserResourceKind: "role",
		mUtils.DbResourceKind:   "keyspace",
	}

	listKeyspacePermissions = "LIST ALL PERMISSIONS on keyspace %s"
	roleKeyColumn           = "role"

	roleToPermissions = map[string][]string{
		"admin": {"ALL"},
		"rw":    {"MODIFY", "SELECT"},
		"ro":    {"SELECT"},
	}
)

type CassandraDbAdministration struct {
	logger            *zap.Logger
	supportedRoles    []string
	supportedFeatures map[string]bool
	apiVersion        string
	cassandraService  cassandra.CassandraService
	sessionService    cassandra.SessionService
}

func GetCassandraDbAdministration(
	logger *zap.Logger,
	supportedRoles []string,
	supportedFeatures map[string]bool,
	apiVersion,
	hostname string,
	port int,
	user string,
	pass string,
	defaultKeyspace string,
	defaultConsistency gocql.Consistency,
	connectTimeout int,
	timeout int, defaultTopology string,
	streamingRoleName string, streamingRolePermissions []string) service.DbAdministration {

	roleToPermissions[streamingRoleName] = streamingRolePermissions

	sessionService := &cassandra.SessionServiceImpl{
		Logger:             logger,
		DefaultKeyspace:    defaultKeyspace,
		DefaultConsistency: defaultConsistency,
		Configuration: &cassandra.CassandraConfigurationImpl{
			Hostname:       hostname,
			Port:           port,
			User:           user,
			Pass:           pass,
			ConnectTimeout: connectTimeout,
			Timeout:        timeout,
			Topology:       defaultTopology,
		},
	}
	return &CassandraDbAdministration{
		logger:            logger,
		supportedRoles:    supportedRoles,
		supportedFeatures: supportedFeatures,
		apiVersion:        apiVersion,
		sessionService:    sessionService,
		cassandraService: &cassandra.CassandraServiceImpl{
			Logger: logger,
		},
	}
}

var _ service.DbAdministration = &CassandraDbAdministration{}

func (c *CassandraDbAdministration) CreateRoles(ctx context.Context, roles []dao.AdditionalRole) (success []dao.Success, failure *dao.Failure) {
	logger := utils.AddLoggerContext(c.logger, ctx)

	var additionalRoleIdInProcess string

	defer func() {
		if r := recover(); r != nil {
			logger.Error(fmt.Sprintf("error during additional roles creation %s", r))
			if failure == nil {
				failure = &dao.Failure{
					Id:      additionalRoleIdInProcess,
					Message: fmt.Sprintf("%s", r),
				}
			}
		}
	}()

	for _, additionalRole := range roles {
		existingRoles := make([]string, 0)
		additionalRoleIdInProcess = additionalRole.Id
		dbName := additionalRole.DbName

		for _, connectionProperties := range additionalRole.ConnectionProperties {
			roleForCheck := connectionProperties["role"].(string) //TODO
			existingRoles = append(existingRoles, roleForCheck)
		}
		newConProps := make([]dao.ConnectionProperties, 0)
		newResources := make([]dao.DbResource, 0)
		for _, role := range c.GetSupportedRoles() {
			if !Contains(existingRoles, role) {
				userCreateRequest := dao.UserCreateRequest{
					DbName: dbName,
					Role:   role,
				}

				createdUser, err := c.CreateUser(ctx, "", userCreateRequest)
				if err != nil {
					failure = &dao.Failure{
						Id:      additionalRoleIdInProcess,
						Message: err.Error(),
					}
					break
				}

				newConProps = append(newConProps, createdUser.ConnectionProperties)
				newResources = append(newResources, createdUser.Resources...)
			}
		}

		success = append(success, dao.Success{
			Id:                   additionalRole.Id,
			ConnectionProperties: newConProps,
			Resources:            newResources,
			DbName:               dbName,
		})
	}

	return success, failure
}

func (c *CassandraDbAdministration) GetFeatures() map[string]bool {
	return c.supportedFeatures
}

func (c *CassandraDbAdministration) GetSupportedRoles() []string {
	if c.supportedFeatures[mUtils.FeatureMultiUsers] {
		return c.supportedRoles
	}
	return []string{"admin"}
}

func (c *CassandraDbAdministration) GetVersion() dao.ApiVersion {
	return dao.ApiVersion(c.apiVersion)
}

func (c *CassandraDbAdministration) validateRequestParamsAndGetLogicalDbName(requestOnCreateDb dao.DbCreateRequest) (string, error) {

	name := ""

	if requestOnCreateDb.NamePrefix != nil {
		if len(*requestOnCreateDb.NamePrefix) > 0 {
			if !prefixRegexp.MatchString(*requestOnCreateDb.NamePrefix) {
				return "", &utils.ExecutionError{Msg: "namePrefix must not contain the following characters :/'\\\"%?.,@#&*()"}
			} else {
				// Prefix delimiter is not set due to previous java implementation compatibility
				//name = *requestOnCreateDb.NamePrefix + c.GetDBPrefixDelimiter()
				name = *requestOnCreateDb.NamePrefix
			}
		}
	} else {
		if classifier, ok := requestOnCreateDb.Metadata["classifier"].(map[string]interface{}); ok {
			if namespace, ok := classifier["namespace"].(string); ok {
				if microserviceName, ok := classifier["microserviceName"].(string); ok {
					shrinkedName, err := utils.PrepareDatabaseName(namespace, microserviceName, 48)
					if err != nil {
						panic(err)
					}
					return strings.ReplaceAll(shrinkedName, "-", "_"), nil
				}
			}
		} else if c.GetDBPrefix() != "" {
			name = c.GetDBPrefix() + c.GetDBPrefixDelimiter()
		}
	}

	if !dbNameRegexp.MatchString(requestOnCreateDb.DbName) {
		return "", &utils.ExecutionError{Msg: "dbName must not contain the following characters :/'\\\"%?.,@#&*()"}
	}

	if !userNameRegexp.MatchString(requestOnCreateDb.Username) {
		return "", &utils.ExecutionError{Msg: "userName must not contain the following characters :/'\\\"%?.,@#&*()"}
	}

	return name + requestOnCreateDb.DbName, nil
}

func (c *CassandraDbAdministration) getKeySpaceConnectionProperties(keyspaceName, username, password, role string) dao.ConnectionProperties {

	connectionProps := dao.ConnectionProperties{
		"url": fmt.Sprintf("cassandra://%v:%v/%v",
			c.sessionService.GetConfiguration().GetHost(),
			c.sessionService.GetConfiguration().GetPort(),
			keyspaceName),
		"port":          c.sessionService.GetConfiguration().GetPort(),
		"username":      username,
		"password":      password,
		"keyspace":      keyspaceName,
		"contactPoints": []string{c.sessionService.GetConfiguration().GetHost()},
		"tls":           c.sessionService.GetConfiguration().IsTLSEnabled(),
	}
	if len(role) > 0 && c.GetVersion() != "v1" {
		connectionProps["role"] = role
	}

	return connectionProps
}

func (c *CassandraDbAdministration) UpdateCassandraSettingsHandler(ctx *fiber.Ctx) error {
	logger := utils.AddLoggerContext(c.logger, context.Background())
	logger.Info("Received request to update settings")
	dbName := ctx.Params("dbName")

	if !mUtils.ValidateDbIdentifierParam(context.Background(), "dbName", dbName, dbNameRegexpExpression) {
		return mUtils.SendInvalidParameterResponse(ctx, "dbName", dbName, dbNameRegexpExpression)
	}

	var updateSettingsRequest cassandra.CassandraUpdateSettingsRequest
	err := ctx.BodyParser(&updateSettingsRequest)
	if err != nil {
		logger.Error("Failed to parse request in update settings handler", zap.Error(err))
		return ctx.Status(500).SendString(err.Error())
	}
	newReplication, okNew := updateSettingsRequest.NewSettings["replication"].(string)
	if okNew {
		c.sessionService.NewAutoCloseSession(func(sessionInterface cassandra.Session) interface{} {
			// Ensure metadata table exists
			if err := c.cassandraService.EnsureMetadataTable(ctx.Context(), sessionInterface, dbName); err != nil {
				logger.Error("Failed to ensure metadata table exists", zap.Error(err))
				return ctx.Status(500).SendString(err.Error())
			}

			// Upsert replication setting
			if err := c.cassandraService.UpsertMetadataSetting(ctx.Context(), sessionInterface, dbName, "replication", newReplication); err != nil {
				logger.Error("Failed to upsert replication metadata", zap.Error(err))
				return ctx.Status(500).SendString(err.Error())
			}

			logger.Info("Replication settings successfully updated in metadata table")
			return nil
		})
	} else {
		logger.Warn("Replication key missing or invalid in new settings")
	}

	return ctx.Status(200).SendString("Update settings processed successfully")
}

func (c *CassandraDbAdministration) CreateDatabase(ctx context.Context, requestOnCreateDb dao.DbCreateRequest) (string, *dao.LogicalDatabaseDescribed, error) {
	logger := utils.AddLoggerContext(c.logger, ctx)

	logicalDatabaseName, validationError := c.validateRequestParamsAndGetLogicalDbName(requestOnCreateDb)
	if validationError != nil {
		return "", nil, validationError
	}

	var connectionProps []dao.ConnectionProperties
	var resources []dao.DbResource

	c.sessionService.NewAutoCloseSession(func(sessionInterface cassandra.Session) interface{} {
		defer func() {
			if r := recover(); r != nil {
				logger.Info(fmt.Sprintf("Failed to create database: %v", r))
				logger.Info("Rollback. Dropping created resources")
				errMsg := fmt.Sprintf("Recovered with %v\n", r)
				if err := c.cassandraService.DropResource(ctx, sessionInterface, "keyspace", logicalDatabaseName); err != nil {
					errMsg = errMsg + fmt.Sprintf("Rollback failed to drop keyspace with %v", err)
				}
				if err := c.cassandraService.DropResource(ctx, sessionInterface, "role", requestOnCreateDb.Username); err != nil {
					errMsg = errMsg + fmt.Sprintf("Rollback failed to drop role with %v", err)
				}
				panic(errMsg)
			}
		}()

		createKeyspaceErr := c.cassandraService.CreateKeyspace(ctx, sessionInterface, logicalDatabaseName, requestOnCreateDb.Settings[replicationKey].(string))
		if createKeyspaceErr != nil {
			panic(createKeyspaceErr)
		}
		createTableErr := c.cassandraService.CreateTable(ctx, sessionInterface, logicalDatabaseName, "metadata")
		if createTableErr != nil {
			panic(createTableErr)
		}

		marshaledMeta, err := json.Marshal(requestOnCreateDb.Metadata)
		utils.PanicError(err, c.logger.Error, "Failed marshaling metadata for keyspace")
		sessionInterface.Query(
			fmt.Sprintf("insert into %s.metadata (id, value) values (?, ?)", logicalDatabaseName),
			mUtils.MetadataKey,
			string(marshaledMeta)).Exec(true)

		logger.Debug(fmt.Sprintf("Created metadata for %s keyspace", logicalDatabaseName))
		replicationValue, ok := requestOnCreateDb.Settings[replicationKey].(string)
		if ok {
			sessionInterface.Query(
				fmt.Sprintf("INSERT INTO %s.metadata (id, value) VALUES (?, ?)", logicalDatabaseName),
				"replication",
				strings.TrimSpace(replicationValue),
			).Exec(true)
		}

		logger.Debug(fmt.Sprintf("Added settings for %s keyspace", logicalDatabaseName))
		resources = append(resources, dao.DbResource{
			Kind: mUtils.DbResourceKind,
			Name: logicalDatabaseName,
		})

		if c.apiVersion == "v1" {
			if err := c.cassandraService.CreateRole(ctx, sessionInterface, requestOnCreateDb.Username, requestOnCreateDb.Password); err != nil {
				panic(err)
			}
			if err := c.cassandraService.GrantPermissions(ctx, sessionInterface, logicalDatabaseName, requestOnCreateDb.Username, []string{"ALL"}); err != nil {
				panic(err)
			}
			connectionProps = append(connectionProps, c.getKeySpaceConnectionProperties(
				logicalDatabaseName, requestOnCreateDb.Username, requestOnCreateDb.Password, ""))
			resources = append(resources, dao.DbResource{
				Kind: mUtils.UserResourceKind,
				Name: requestOnCreateDb.Username,
			})
		} else {
			for _, role := range c.GetSupportedRoles() {
				var username string
				var password string
				if role == "admin" {
					if requestOnCreateDb.Username != "" {
						username = requestOnCreateDb.Username
					}
					if requestOnCreateDb.Password != "" {
						password = requestOnCreateDb.Password
					}
				} else {
					username = fmt.Sprintf("dbaas_%s", c.generateUUID())
					password = c.generateUUID()
				}

				if err := c.cassandraService.CreateRole(ctx, sessionInterface, username, password); err != nil {
					panic(err)
				}
				if err := c.cassandraService.GrantPermissions(ctx, sessionInterface, logicalDatabaseName, username, roleToPermissions[role]); err != nil {
					panic(err)
				}
				connectionProps = append(connectionProps, c.getKeySpaceConnectionProperties(
					logicalDatabaseName, username, password, role))
				resources = append(resources, dao.DbResource{
					Kind: mUtils.UserResourceKind,
					Name: username,
				})
			}
		}
		return nil
	})

	return logicalDatabaseName,
		&dao.LogicalDatabaseDescribed{
			ConnectionProperties: connectionProps,
			Resources:            resources,
		},
		nil
}

func (c *CassandraDbAdministration) DescribeDatabases(ctx context.Context, logicalDatabases []string, showResources bool, showConnections bool) map[string]dao.LogicalDatabaseDescribed {
	panic("Describe is not supported")
}

func (c *CassandraDbAdministration) GetDatabases(ctx context.Context) []string {
	return c.sessionService.NewAutoCloseSession(func(sessionInterface cassandra.Session) interface{} {
		var dbsList []string
		dbsListIterator := sessionInterface.Query(
			"SELECT keyspace_name FROM system_schema.keyspaces").Iter()
		var dbName string
		for dbsListIterator.Scan(&dbName) {
			dbsList = append(dbsList, dbName)
		}
		defer dbsListIterator.Close()
		return dbsList
	}).([]string)
}

func (c *CassandraDbAdministration) DropResources(ctx context.Context, resources []dao.DbResource) []dao.DbResource {
	return c.sessionService.NewAutoCloseSession(func(sessionInterface cassandra.Session) interface{} {
		var dropStatuses []dao.DbResource
		catchDropFailing := func(src dao.DbResource) (resource dao.DbResource) {
			resource = src
			defer func() {
				if r := recover(); r != nil {
					resource.Status = dao.DELETE_FAILED
					resource.ErrorMessage = fmt.Sprintf("%v", r)
				}
			}()
			if err := c.cassandraService.DropResource(ctx, sessionInterface, apiResourcesToCassandraEntities[resource.Kind], resource.Name); err != nil {
				panic(err)
			}
			resource.Status = dao.DELETED
			return resource
		}
		for _, resource := range resources {
			dropStatuses = append(dropStatuses, catchDropFailing(resource))
		}
		return dropStatuses
	}).([]dao.DbResource)
}

func (c *CassandraDbAdministration) GetMetadata(ctx context.Context, logicalDatabase string) map[string]interface{} {
	metadata := c.sessionService.NewAutoCloseSession(func(sessionInterface cassandra.Session) interface{} {
		dbsListIterator := sessionInterface.Query(fmt.Sprintf("SELECT value FROM %s.metadata", logicalDatabase)).Iter()
		var metadata string
		var result map[string]interface{}
		defer dbsListIterator.Close()
		if dbsListIterator.Scan(&metadata) {
			if err := json.Unmarshal([]byte(metadata), &result); err == nil {
				return result
			} else {
				return nil
			}
		} else {
			return nil
		}
	})
	if metadata != nil {
		return metadata.(map[string]interface{})
	} else {
		c.logger.Error(fmt.Sprintf("can't get metadata for %s", logicalDatabase))
		return nil
	}
}

func (c *CassandraDbAdministration) UpdateMetadata(ctx context.Context, newMetadata map[string]interface{}, serviceName string) {
	c.sessionService.NewAutoCloseSession(func(sessionInterface cassandra.Session) interface{} {
		marshaledMeta, err := json.Marshal(newMetadata)
		utils.PanicError(err, c.logger.Error, "Failed marshaling metadata for keyspace")
		sessionInterface.Query(
			fmt.Sprintf("update %s.metadata set value = ? where id = ?", serviceName),
			string(marshaledMeta), mUtils.MetadataKey).Exec(true)

		return nil
	})
}

func (c *CassandraDbAdministration) generateUUID() string {
	return utils.Substr(uuid.New().String(), 0, 7)
}

func (c *CassandraDbAdministration) generateDBName() string {
	currentTime := time.Now().UTC()
	timestamp := currentTime.Format("150405.000.020106")
	return strings.ReplaceAll(timestamp, ".", "")
}

func (c *CassandraDbAdministration) generateUserName() string {
	return "role_" + c.generateUUID()
}

func (c *CassandraDbAdministration) GetDefaultCreateRequest() dao.DbCreateRequest {
	return dao.DbCreateRequest{
		Metadata: map[string]interface{}{},
		Password: c.generateUUID(),
		DbName:   c.generateDBName(),
		Settings: map[string]interface{}{
			replicationKey: c.sessionService.GetConfiguration().GetDefaultTopology(),
		},
		Username: c.generateUserName(),
	}
}

func (c *CassandraDbAdministration) GetDefaultUserCreateRequest() dao.UserCreateRequest {
	return dao.UserCreateRequest{
		DbName:   "",
		Password: c.generateUUID(),
	}
}

func (c *CassandraDbAdministration) PreStart() {}

func (c *CassandraDbAdministration) CreateUser(ctx context.Context, userName string, requestOnCreateUser dao.UserCreateRequest) (*dao.CreatedUser, error) {
	logger := utils.AddLoggerContext(c.logger, ctx)
	// TODO v2 move validations some place else
	if c.GetVersion() == "v2" && !Contains(c.GetSupportedRoles(), requestOnCreateUser.Role) {
		logger.Error(fmt.Sprintf("role '%s' is not supported. Only roles: %s", requestOnCreateUser.Role, c.GetSupportedRoles()))
	}

	if requestOnCreateUser.Role == "" {
		requestOnCreateUser.Role = "admin"
	}

	if userName == "" {
		userName = c.generateUserName()
	}

	if requestOnCreateUser.Password == "" {
		requestOnCreateUser.Password = c.generateUUID()
	}

	if !userNameRegexp.MatchString(userName) {
		return nil, &utils.ExecutionError{Msg: "userName must not contain the following characters :/'\\\"%?.,@#&*()"}
	}

	return c.sessionService.NewAutoCloseSession(func(sessionInterface cassandra.Session) interface{} {
		resources := []dao.DbResource{
			{
				Kind: mUtils.UserResourceKind,
				Name: userName,
			},
		}

		roles, err := c.cassandraService.GetAllRoles(ctx, sessionInterface)
		if err != nil {
			panic(err)
		}
		roleExists := false
		for role, _ := range roles {
			if role == userName {
				roleExists = true
				logger.Debug(fmt.Sprintf("Found existing %s role", userName))
				break
			}
		}

		if roleExists {
			// Prevent updating for role in case it doesn't have permissions for keyspace
			if requestOnCreateUser.DbName != "" {
				hasPermissions := false
				keySpaceRoles, err := c.cassandraService.GetRolesForKeyspace(ctx, sessionInterface, requestOnCreateUser.DbName)
				if err != nil {
					panic(err)
				}
				for role, _ := range keySpaceRoles {
					if hasPermissions = role == userName; hasPermissions {
						break
					}
				}
				if !hasPermissions {
					msg := fmt.Sprintf("Not found existing %s role permissions for %s keyspace", userName, requestOnCreateUser.DbName)
					utils.PanicError(
						&utils.ExecutionError{Msg: msg},
						logger.Error, msg)
				}
			}

			logger.Debug(fmt.Sprintf("Updating %s role password", userName))
			sessionInterface.Query(
				fmt.Sprintf("ALTER ROLE '%s' WITH PASSWORD = '%s' AND LOGIN = true",
					userName, requestOnCreateUser.Password)).Exec(true)
		} else {
			if err := c.cassandraService.CreateRole(ctx, sessionInterface, userName, requestOnCreateUser.Password); err != nil {
				panic(err)
			}
			if err := c.cassandraService.GrantPermissions(ctx, sessionInterface, requestOnCreateUser.DbName, userName, roleToPermissions[requestOnCreateUser.Role]); err != nil {
				panic(err)
			}
		}

		return &dao.CreatedUser{
			ConnectionProperties: c.getKeySpaceConnectionProperties(
				requestOnCreateUser.DbName,
				userName,
				requestOnCreateUser.Password,
				requestOnCreateUser.Role),
			Resources: resources,
			Name:      requestOnCreateUser.DbName,
		}
	}).(*dao.CreatedUser), nil
}

func (c *CassandraDbAdministration) MigrateToVault(ctx context.Context, dbName, userName string) error {
	// No additional actions needed
	return nil
}

func (c *CassandraDbAdministration) GetDBPrefix() string {
	return "dbaas"
}

func (c *CassandraDbAdministration) GetDBPrefixDelimiter() string {
	return "_"
}

func Contains(slice []string, element string) bool {
	for _, v := range slice {
		if v == element {
			return true
		}
	}
	return false
}
