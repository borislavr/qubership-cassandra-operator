*** Variables ***
${CASSANDRA_HOST}                                 %{CASSANDRA_HOST}
${CASSANDRA_USERNAME}                             %{CASSANDRA_USERNAME}
${CASSANDRA_PASSWORD}                             %{CASSANDRA_PASSWORD}
${CASSANDRA_NAMESPACE}                            %{NAMESPACE}
${TEST_KEYSPACES_REPLICATION_FACTOR}              %{TEST_KEYSPACES_REPLICATION_FACTOR}
${DC_NAME}                                        %{DC_NAME}
${WAIT_TIMEOUT}                                   %{WAIT_TIMEOUT}
${TLS_ENABLED}                                    %{TLS_ENABLED=false}

*** Settings ***
Library  String
Library	 Collections
Library	 RequestsLibrary
Library           ../lib/CassandraLibrary.py	${CASSANDRA_HOST}  ${CASSANDRA_USERNAME}  ${CASSANDRA_PASSWORD}  ${WAIT_TIMEOUT}
Library  PlatformLibrary  managed_by_operator=true

*** Keywords ***
Prepare Shared
    &{headers}=  Create Dictionary  Content-Type=application/json  Accept=application/json
    Set Suite Variable  ${headers}
    Test Cassandra connection

Test Cassandra connection
	Connect To Cassandra  ${TLS_ENABLED}

Insert To ${table} And Check
    ${col1}=  Evaluate    int(random.random()*1000)    random
    ${col2}=  Generate Random String  10
    Set Suite Variable  ${col1}
    Set Suite Variable  ${col2}
    Insert To Table    ${col1}  ${col2}  ${table}
    ${result}=  Select From Table  ${table}
    Log  ${result}
    Should Contain  ${result}  ${col1}
    Should Contain  ${result}  ${col2}

Delete From ${table} And Check
    Delete From Table    ${col1}  ${table}
    ${result}=  Select From Table  ${table}
    Log  ${result}
    Should Not Contain  ${result}  ${col1}

Check Data In Table
    [Arguments]  ${keyspace}  ${int_col}=${col1}  ${text_col}=${col2}
    ${result}=  Select From Table  ${keyspace}
    Log  ${result}
    Should Contain  ${result}  ${int_col}
    Should Contain  ${result}  ${text_col}

Check Data Not Exists In ${table}
    ${result}=  Select From Table  ${table}
    Log  ${result}
    Should Not Contain  ${result}  ${col1}
    Should Not Contain  ${result}  ${col2}

Check Error In Jobstatus
    [Arguments]  ${job}  ${error_text}  ${status_code}
    ${resp}=  GET On Session  backupsession  /jobstatus/${job}  expected_status=${status_code}
    Log  ${resp.content}
    Should Be True  """${error_text}""" in """${resp.content}"""

Create Data
    [Arguments]  ${keyspace_name}
    Create Keyspace  ${keyspace_name}  ${DC_NAME}  ${TEST_KEYSPACES_REPLICATION_FACTOR}
    Create Table  ${keyspace_name}
    Insert To ${keyspace_name} And Check

Create Empty Keyspace
    [Arguments]  ${keyspace_name}
    Create Keyspace  ${keyspace_name}  ${DC_NAME}  ${TEST_KEYSPACES_REPLICATION_FACTOR}

Delete Keyspace ${keyspace_name} And Check
    ${res}=  Get All Keyspaces
    Should Contain  ${res}  ${keyspace_name}
    Delete Keyspace  ${keyspace_name}
    ${res}=  Get All Keyspaces
    Should Not Contain  ${res}  ${keyspace_name}

Check ${keyspace_name} exists
    ${res}=  Get All Keyspaces
    Should Contain  ${res}  ${keyspace_name}

Get Pod Name For Deployment
    [Arguments]  ${deployment}
    @{pods}=  Get Pod Names For Deployment Entity  ${deployment}  ${CASSANDRA_NAMESPACE}
    ${pod_name}  Set Variable  ${pods}[0]
    [Return]  ${pod_name}

Check Backup Exists Using Baskup-daemon Pod
    [Arguments]  ${backup_daemon_pod}  ${backup}
    ${backups}=  Execute Command In Pod  ${backup_daemon_pod}  ${CASSANDRA_NAMESPACE}  cd backup-storage/granular/ && ls
    Should Be True  """${backup}""" in """${backups}[0]"""

Check Pod Status Is Running
    [Arguments]  ${pod_name}
    ${pod}=  Get Pod  ${pod_name}  ${CASSANDRA_NAMESPACE}
    Should Be Equal As Strings  ${pod.status.phase}  Running

Fetch Cassandra Version
    ${version}=  Get Cassandra Version
    [Return]  ${version}