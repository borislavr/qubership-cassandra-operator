package cassandra

import (
	"context"
	"fmt"

	"github.com/Netcracker/qubership-dbaas-adapter-core/pkg/utils"
	"go.uber.org/zap"
)

type CassandraService interface {
	GrantPermissions(ctx context.Context, session Session, dbName, role string, permissions []string) error
	CreateRole(ctx context.Context, session Session, role, password string) error
	CreateKeyspace(ctx context.Context, session Session, dbName, settings string) error
	CreateTable(ctx context.Context, session Session, dbName, tableName string) error
	GetRolesForKeyspace(ctx context.Context, session Session, keyspace string) (map[string]bool, error)
	GetAllRoles(ctx context.Context, session Session) (map[string]bool, error)
	DropResource(ctx context.Context, session Session, resourceKind, name string) error
}

type CassandraServiceImpl struct {
	Logger *zap.Logger
}

var _ CassandraService = &CassandraServiceImpl{}

func (r *CassandraServiceImpl) executeStatement(ctx context.Context, session Session, stmt string) error {
	logger := utils.AddLoggerContext(r.Logger, ctx)
	logger.Debug(stmt)

	err := session.Query(stmt).Exec(false)
	return err
}

func (r *CassandraServiceImpl) DropResource(ctx context.Context, session Session, resourceKind, name string) error {
	return r.executeStatement(ctx, session, fmt.Sprintf("drop %s if exists %s", resourceKind, name))
}

func (r *CassandraServiceImpl) CreateKeyspace(ctx context.Context, session Session, dbName, settings string) error {
	return r.executeStatement(ctx, session, fmt.Sprintf("create KEYSPACE %s WITH REPLICATION = %s",
		dbName, settings))
}

func (r *CassandraServiceImpl) CreateTable(ctx context.Context, session Session, dbName, tableName string) error {
	return r.executeStatement(ctx, session, fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s.%s(id text PRIMARY KEY, value text)", dbName, tableName))
}

func (r *CassandraServiceImpl) CreateRole(ctx context.Context, session Session, role, password string) error {
	return r.executeStatement(ctx, session, fmt.Sprintf("CREATE ROLE IF NOT EXISTS '%s' WITH PASSWORD = '%s' AND LOGIN = true", role, password))
}

func (r *CassandraServiceImpl) GrantPermissions(ctx context.Context, session Session, dbName, role string, permissions []string) error {
	logger := utils.AddLoggerContext(r.Logger, ctx)
	for _, permission := range permissions {
		stmt := fmt.Sprintf("GRANT %s on KEYSPACE %s to '%s'", permission, dbName, role)
		logger.Debug(stmt)
		err := session.Query(stmt).Exec(false)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *CassandraServiceImpl) GetRolesForKeyspace(ctx context.Context, session Session, keyspace string) (map[string]bool, error) {
	stmt := fmt.Sprintf("LIST ALL PERMISSIONS on keyspace %s", keyspace)
	return r.getRoles(ctx, session, stmt)
}
func (r *CassandraServiceImpl) GetAllRoles(ctx context.Context, session Session) (map[string]bool, error) {
	stmt := "LIST ROLES"
	return r.getRoles(ctx, session, stmt)
}

func (r *CassandraServiceImpl) getRoles(ctx context.Context, session Session, stmt string) (map[string]bool, error) {
	logger := utils.AddLoggerContext(r.Logger, ctx)
	logger.Debug(stmt)

	keyspaceRolesIterator := session.Query(stmt).Iter()
	roles := map[string]bool{}
	rd, err := keyspaceRolesIterator.RowData()
	if err != nil {
		return nil, err
	}
	for keyspaceRolesIterator.Scan(rd.RowData.Values...) {
		if iRole := rd.GetValue("role"); iRole != nil {
			roles[*iRole.(*string)] = true
		}
	}
	keyspaceRolesIterator.Close()

	return roles, nil
}
