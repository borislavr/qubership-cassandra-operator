package utils //todo package name confuses

const Cassandra = "cassandra"
const CassandraCluster = "cassandra-cluster"

const KubeHostName = "kubernetes.io/hostname"
const ContextPasswordKey = "ctxPasswordKey"
const PVNodesFormat = "pvNodeNames%v"

const KubernetesHelperImpl = "kubernetesHelperImpl"
const ContextClusterBuilder = "clusterBuilder"

const ContextCredsManager = "contextCredsManager"

const Name = "name"
const Service = "service"
const App = "app"
const Username = "username"
const Password = "password"
const Roles = "roles"

const SSHSecret = "ssh-keys"

const Microservice = "microservice"

const TriesCount = "triesCount"
const RetryTimeoutSec = "retryTimeout"

const BackupPvcName = "backup-data-%v"
const Backup = "backup"

const BackupDaemon = "cassandra-backup-daemon"
const BackupStorage = "backup-storage"

var BackupEntrypoint = []string{"/opt/backup/run.sh"}

const Robot = "robot-tests"

const Dbaas = "dbaas"

const DbaasName = "dbaas-cassandra-adapter"

const RootCert = "root-ca"
const RootCertPath = "/usr/ssl/"

const ServerCertsPath = "/certs/"

const AccessKey = "accessKey"
const SecretKey = "secretKey"
const Region = "region"

var RobotEntrypoint = []string{"/docker-entrypoint.sh"}

const Charset = "abcdefghijklmnopqrstuvwxyz" +
	"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

const DbaasAdminRoleCreds = "dbaas-streaming-role"

// labels
const (
	AppName              = "app.kubernetes.io/name"
	AppInstance          = "app.kubernetes.io/instance"
	AppVersion           = "app.kubernetes.io/version"
	AppComponent         = "app.kubernetes.io/component"
	AppManagedBy         = "app.kubernetes.io/managed-by"
	AppManagedByOperator = "app.kubernetes.io/managed-by-operator"
	AppProcByOperator    = "app.kubernetes.io/processed-by-operator"
	AppTechnology        = "app.kubernetes.io/technology"
	AppPartOf            = "app.kubernetes.io/part-of"
)
