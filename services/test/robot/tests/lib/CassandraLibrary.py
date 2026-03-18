from cassandra import ConsistencyLevel
from cassandra.auth import PlainTextAuthProvider
from cassandra.cluster import Cluster, ExecutionProfile, EXEC_PROFILE_DEFAULT
from cassandra.policies import WhiteListRoundRobinPolicy, DowngradingConsistencyRetryPolicy
from cassandra.cluster import Cluster
from robot.api import logger
from ssl import PROTOCOL_TLSv1_2, SSLContext, PROTOCOL_TLSv1, CERT_REQUIRED
import os
import requests
from robot.libraries.BuiltIn import BuiltIn

def main():
    lib = CassandraLibrary('127.0.0.1')
    lib.connect_to_cassandra()
    lib.create_keyspace(dc_name="dc1", replication_factor=3)
    lib.create_table()
    lib.insert_to_table(123, "321")
    lib.insert_to_table(111, "321")
    lib.insert_to_table(222, "321")
    lib.select_from_table()
    lib.delete_from_table(111)
    lib.delete_keyspace()
    lib.get_cassandra_version()


class CassandraLibrary(object):
    """
    Cassandra testing library for Robot Framework.
    """

    ROBOT_LIBRARY_SCOPE = 'GLOBAL'

    def __init__(self, host, username="admin", password="admin", req_timeout=50.0, port=9042,
                 consistency_level=ConsistencyLevel.ONE):
        self.host = [host]
        self.port = int(port)
        self.username = username
        self.password = password
        self.timeout = int(req_timeout) / 5
        self.session = None
        self.consistency_level = consistency_level

    def connect_to_cassandra(self, tls_enabled=False):
        logger.debug(self.timeout)
        auth_provider = PlainTextAuthProvider(
            username=self.username, password=self.password)
        profile = ExecutionProfile(
            consistency_level=self.consistency_level,
            request_timeout=self.timeout,
        )
        ssl_context = None
        if tls_enabled:
            ssl_context = SSLContext(PROTOCOL_TLSv1_2)
            ssl_context.load_verify_locations(os.getenv("TLS_ROOTCERT", "/opt/ca.crt"))
            ssl_context.verify_mode = CERT_REQUIRED

        cluster = Cluster(self.host, execution_profiles={
                EXEC_PROFILE_DEFAULT: profile}, auth_provider=auth_provider, ssl_context=ssl_context)
        self.session = cluster.connect()

    def create_keyspace(self, keyspace="test_keyspace", dc_name="main", replication_factor=1):
        logger.debug('keyspace: ' + keyspace)
        logger.debug('replication_factor: ' + str(replication_factor))
        logger.debug('dc_name: ' + dc_name)
        self.session.execute("""
                CREATE KEYSPACE IF NOT EXISTS %s
                WITH replication = { 'class': 'NetworkTopologyStrategy', '%s': '%d' }
                """ % (keyspace, dc_name, replication_factor))

    def create_table(self, keyspace="test_keyspace", table="test_table"):
        self.session.execute("""
                CREATE TABLE IF NOT EXISTS %s.%s (
                    col1 int,
                    col2 text,
                    PRIMARY KEY (col1)
                )
                """ % (keyspace, table))

    def insert_to_table(self, col1, col2, keyspace="test_keyspace", table="test_table"):
        prepared = self.session.prepare("""
                    INSERT INTO %s.%s (col1, col2)
                    VALUES (?, ?)
                    """ % (keyspace, table))

        self.session.execute(prepared, (col1, col2))

    def select_from_table(self, keyspace="test_keyspace", table="test_table"):
        future = self.session.execute_async(
            "SELECT * FROM %s.%s" % (keyspace, table))
        try:
            rows = future.result()
        except Exception as e:
            logger.error(f"Error reading rows:{e}")
            return

        result = []
        for row in rows:
            result.append(row[0])
            result.append(row[1])

        return result

    def delete_from_table(self, key, keyspace="test_keyspace", table="test_table"):
        self.session.execute("delete from %s.%s where col1=%d" %
                             (keyspace, table, key))

    def delete_keyspace(self, keyspace="test_keyspace"):
        self.session.execute("DROP KEYSPACE IF EXISTS %s" % keyspace)

    def get_all_keyspaces(self):
        keyspace_list = []
        for i in self.session.execute("SELECT * FROM system_schema.keyspaces;"): keyspace_list.append(i.keyspace_name)
        return keyspace_list

    def get_multiple_users_name(self, connectionProperties):
        users_roles = {}
        for con in connectionProperties:
            users_roles[con['role']] = con['username']
        return users_roles

    def get_permission_for_role(self, role="dbaas_test"):
        future = self.session.execute_async("list all of %s" % role)
        try:
            rows = future.result()
        except Exception as e:
            logger.error(f"Error reading rows:{e}")
            return
        result = []
        for row in rows:
            result.append(row[3])
        return result

    def get_all_users(self):
        all_users = self.session.execute_async("LIST USERS")
        try:
            rows = all_users.result()
        except Exception:
            logger.error("Error reading rows:")
            return
        users = []
        for row in rows:
            users.append(row[0])
        return users

    def delete_user(self, user):
        self.session.execute("DROP USER IF EXISTS %s" % user)

    def revoke_permission_from_user(self, permission, resource, user):
        self.session.execute("REVOKE %s on %s FROM %s" % (permission, resource, user))

    def get_cassandra_version(self):
        query = "SELECT release_version FROM system.local"
        result = self.session.execute(query)
        version = result.one().release_version
        return version


if __name__ == "__main__":
    main()
