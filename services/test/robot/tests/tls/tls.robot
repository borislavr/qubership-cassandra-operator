*** Variables ***
${BACKUP_HOST}                                    %{BACKUP_HOST}
${BACKUP_DAEMON_API_CREDENTIALS_USERNAME}         %{BACKUP_DAEMON_API_CREDENTIALS_USERNAME}
${BACKUP_DAEMON_API_CREDENTIALS_PASSWORD}         %{BACKUP_DAEMON_API_CREDENTIALS_PASSWORD}

*** Settings ***
Library  String
Resource  ../shared/keywords.robot
Resource  ../dbaas/dbaas_shared.robot


*** Test Cases ***
Check Connection To TLS Cassandra Without Cert
    [Tags]  tls  cassandra
    Skip If  "${TLS_ENABLED}" == "false"  TLS_ENABLED = False, not possible to check case!
    Connect To Cassandra  tls_enabled=${True}
    Run Keyword And Expect Error  *NoHostAvailable*  Connect To Cassandra  tls_enabled=${False}

Check Connection To TLS Dbaas Adapter By HTTP Protocol
    [Tags]  tls  cassandra
    Prepare Configuration For Dbaas Connection
    Skip If  "${TLS_ENABLED}" == "false"  TLS_ENABLED = False, not possible to check case!
    Skip If  "${https_aggregator_enabled}" == "${False}"  HTTP is already active protocol!
    ${verify}=  Get Environment Variable  name=TLS_ROOTCERT  default=False
    ${port}=  Get Environment Variable  name=PORT  default=8080
    Create Session  wrongprotocolsession  http://${DBAAS_ADAPTER_USERNAME}:${DBAAS_ADAPTER_PASSWORD}@${DBAAS_HOST}:${port}  verify=${verify}  timeout=10
    Run Keyword And Expect Error  *ProtocolError*  GET On Session  wrongprotocolsession  /api/${dbaas_api_version}/dbaas/adapter/cassandra/databases

Check Connection To TLS Dbaas Adapter By Not Correct Port
    [Tags]  tls  cassandra
    Skip If  "${TLS_ENABLED}" == "false"  TLS_ENABLED = False, not possible to check case!
    ${verify}=  Get Environment Variable  name=TLS_ROOTCERT  default=False
    Create Session  wrongportsession  https://${DBAAS_ADAPTER_USERNAME}:${DBAAS_ADAPTER_PASSWORD}@${DBAAS_HOST}:8080  verify=${verify}  timeout=10
    Run Keyword And Expect Error  *  GET On Session  wrongportsession  /api/${dbaas_api_version}/dbaas/adapter/cassandra/databases

Check Connection To TLS Backup Daemon By HTTP Protocol
    [Tags]  tls  cassandra
    Skip If  "${TLS_ENABLED}" == "false"  TLS_ENABLED = False, not possible to check case!
    ${verify}=  Get Environment Variable  name=TLS_ROOTCERT  default=False
    ${port}=  Get Environment Variable  name=PORT  default=8080
    Create Session  wrongprotocolsession  http://${BACKUP_DAEMON_API_CREDENTIALS_USERNAME}:${BACKUP_DAEMON_API_CREDENTIALS_PASSWORD}@${BACKUP_HOST}:${port}  verify=${verify}  timeout=10
    Run Keyword And Expect Error  *ProtocolError*  GET On Session  wrongprotocolsession  /backup

Check Connection To TLS Backup Daemon By Not Correct Port
    [Tags]  tls  cassandra
    Skip If  "${TLS_ENABLED}" == "false"  TLS_ENABLED = False, not possible to check case!
    ${verify}=  Get Environment Variable  name=TLS_ROOTCERT  default=False
    Create Session  wrongportsession  https://${BACKUP_DAEMON_API_CREDENTIALS_USERNAME}:${BACKUP_DAEMON_API_CREDENTIALS_PASSWORD}@${BACKUP_HOST}:8080  verify=${verify}  timeout=10
    Run Keyword And Expect Error  *  GET On Session  wrongportsession  /backup