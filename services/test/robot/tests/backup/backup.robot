*** Variables ***
${BACKUP_HOST}                                    %{BACKUP_HOST}
${SYSTEM_KEYSPACE_ERROR_TEXT}                     system keyspace is not user-modifiable
${BACKUP_ERROR_TEXT}                              Internal Server Error
${RESTORE_ERROR_TEXT}                             Wrong db names transferred via API
${ATTEMPTS_NUMBER}                                %{ATTEMPTS_NUMBER=100}

*** Settings ***
Library  String
Library	 Collections
Library	 RequestsLibrary
Library    OperatingSystem
Resource  ../shared/keywords.robot
Suite Setup  Preparation
Suite Teardown  Cleanup

*** Keywords ***
Preparation
    Prepare Shared
    ${CASSANDRA_KEYSPACE}=  Generate Random String  10  [LOWER]
    Set Suite Variable  ${CASSANDRA_KEYSPACE}
    ${GRANULAR_TEST_KEYSPACE}=  Set Variable  granular_${CASSANDRA_KEYSPACE}
    Set Suite Variable  ${GRANULAR_TEST_KEYSPACE}

    ${verify}=    Get Environment Variable    name=TLS_ROOTCERT    default=False
    Set Suite Variable  ${verify}
    ${backup_tls}=    Get Environment Variable    name=TLS_ENABLED    default=False
    ${port}=    Get Environment Variable    name=PORT    default=8080
    Set Suite Variable  ${port}

    ${PROTOCOL} =    Set Variable If    '${backup_tls}' == 'true'
    ...  https
    ...  http
    Set Suite Variable  ${PROTOCOL}

    Create Session    backupsession    ${PROTOCOL}://${BACKUP_DAEMON_API_CREDENTIALS_USERNAME}:${BACKUP_DAEMON_API_CREDENTIALS_PASSWORD}@${BACKUP_HOST}:${port}   verify=${verify}

Cleanup
    DELETE KEYSPACE  ${CASSANDRA_KEYSPACE}
    DELETE KEYSPACE  ${GRANULAR_TEST_KEYSPACE}

Wait For ${job} Job Completion With ${attempts} Attempts
    ${last_index}    Evaluate    ${attempts} - 1
    FOR    ${CheckStatus}    IN RANGE    ${attempts}
        ${resp}=  GET On Session  backupsession  /jobstatus/${job}  expected_status=any
        Log  ${resp.content}
        Exit For Loop If    '${resp.status_code}'=='200'
        Exit For Loop If    '${resp.status_code}'=='400'
        Exit For Loop If    '${resp.status_code}'=='401'
        Exit For Loop If    '${resp.status_code}'=='404'
        Exit For Loop If    '${resp.status_code}'=='500'
        Sleep  5s
        Run Keyword If  ${CheckStatus} == ${last_index}  Fail  Job hasn't finished successfully in ${attempts} attempts, result: ${resp.content}. Increase attempts!
    END
    Log  ${resp}
    Log  ${resp.content}
    RETURN  ${resp}

Run Job To Endpoint And Check
    [Arguments]  ${doc}  ${point}  ${attempts}  ${post_status_code}  ${get_status_code}
    Log  ${post_status_code}
    Log  ${get_status_code}
    ${heads}=  Set Variable  ${None}
    ${heads}=  Run Keyword If  ${doc}  Set Variable  ${headers}

    ${resp}=  POST On Session  backupsession  ${point}  data=${doc}  headers=${heads}  expected_status=${post_status_code}
    Log  ${resp}
    Log  ${resp.content}
    Should Be Equal As Strings  ${resp.status_code}  ${post_status_code}
    ${currbackupjob}=  Set Variable  ${resp.content}

    ${resp}=  Wait For ${currbackupjob} Job Completion With ${attempts} Attempts
    Log  ${resp.content}
    ${status_code}=  Set Variable  ${resp.status_code}
    Set Suite Variable  ${status_code}
    Should Be Equal As Strings  ${resp.status_code}  ${get_status_code}
    RETURN  ${currbackupjob}

Backup Data And Check
    [Arguments]  ${doc}  ${attempts}  ${post_status_code}=200  ${get_status_code}=200
    ${job}=  Run Job To Endpoint And Check  ${doc}  /backup  ${attempts}  ${post_status_code}  ${get_status_code}
    RETURN  ${job}

Restore Data And Check
    [Arguments]  ${doc}  ${attempts}  ${post_status_code}=200  ${get_status_code}=200
    ${job}=  Run Job To Endpoint And Check  ${doc}  /restore  ${attempts}  ${post_status_code}  ${get_status_code}
    RETURN  ${job}

Delete Backup
    [Arguments]  ${backupjob}
    ${resp}=  POST On Session  backupsession  /evict/${backupjob}
    Should Be Equal As Strings  ${resp.status_code}  200

*** Test Cases ***
Test Wrong Backup Credentials
    [Tags]  backup  cassandra
    ${wronguser}=  Generate Random String  10
    ${wrongpass}=  Generate Random String  10
    Create Session    wrongsess    ${PROTOCOL}://${wronguser}:${wrongpass}@${BACKUP_HOST}:${port}    verify=${verify}
    ${resp}=  POST On Session  wrongsess  /backup  expected_status=401
    Log  ${resp}
    Should Be Equal As Strings  ${resp.status_code}  401

Test Empty DB Value
    [Tags]  backup  cassandra
    ${keyspace}=  Set Variable  empty_db_${CASSANDRA_KEYSPACE}
    Create Data  ${keyspace}
    ${document}=  Set Variable  {"dbs":[""]}
    ${backupjob}=  Backup Data And Check  ${document}  ${ATTEMPTS_NUMBER}  post_status_code=500  get_status_code=404
    Check Error In Jobstatus  ${backupjob}  ${BACKUP_ERROR_TEXT}  404
    ${document}=  Set Variable  {"dbs":["${keyspace}"]}
    ${backupjob}=  Backup Data And Check  ${document}  ${ATTEMPTS_NUMBER}
    ${document}=  Set Variable  {"vault":"${backupjob}","dbs":[""]}
    ${restorejob}=  Restore Data And Check  ${document}  ${ATTEMPTS_NUMBER}  post_status_code=500  get_status_code=404
    Check Error In Jobstatus  ${restorejob}  ${RESTORE_ERROR_TEXT}  404
    [Teardown]  Run Keywords  DELETE KEYSPACE  ${keyspace}
    ...  AND  Delete Backup  ${backupjob}

Test Create Granular Data
    [Tags]  backup  cassandra
    Create Data  ${GRANULAR_TEST_KEYSPACE}
    ${granular_backup_col1}=  Set Variable  ${col1}
    ${granular_backup_col2}=  Set Variable  ${col2}
    Set Suite Variable  ${granular_backup_col1}
    Set Suite Variable  ${granular_backup_col2}

Test Backup Granular Data
    [Tags]  backup  cassandra
    ${document}=  Set Variable  {"dbs":["${GRANULAR_TEST_KEYSPACE}"]}
    ${granularbackupjob}=  Backup Data And Check  ${document}  ${ATTEMPTS_NUMBER}
    Set Suite Variable  ${granularbackupjob}

Test Delete Granular Data
    [Tags]  backup  cassandra
    Delete From ${GRANULAR_TEST_KEYSPACE} And Check

Test Restore Granular Data
    [Tags]  backup  cassandra
    ${document}=  Set Variable  {"vault":"${granularbackupjob}","dbs":["${GRANULAR_TEST_KEYSPACE}"]}
    Restore Data And Check  ${document}  ${ATTEMPTS_NUMBER}

Test Check Restored Granular Data
    [Tags]  backup  cassandra
    Check Data In Table  ${GRANULAR_TEST_KEYSPACE}

Test Create Table Level Data
    [Tags]  backup  cassandra
    Create Data  ${CASSANDRA_KEYSPACE}

Test Backup Table Level
    [Tags]  backup  cassandra
    ${document}=  Set Variable  {"dbs":[{"${GRANULAR_TEST_KEYSPACE}": {"tables": ["test_table"]}}]}
    ${granularbackupjob}=  Backup Data And Check  ${document}  ${ATTEMPTS_NUMBER}
    Set Suite Variable  ${granularbackupjob}

Test Delete Table Level Data
    [Tags]  backup  cassandra
    Delete From ${GRANULAR_TEST_KEYSPACE} And Check

Test Restore Table Level
    [Tags]  backup  cassandra
    ${document}=  Set Variable  {"vault":"${granularbackupjob}","dbs":[{"${GRANULAR_TEST_KEYSPACE}": {"tables": ["test_table"]}}]}
    Restore Data And Check  ${document}  ${ATTEMPTS_NUMBER}

Test Create Data
    [Tags]  backup  cassandra
    Create Data  ${CASSANDRA_KEYSPACE}

Test Full Backup Data
	[Tags]  backup  cassandra
    ${backupjob}=  Backup Data And Check  ${None}  ${ATTEMPTS_NUMBER}
    Set Suite Variable  ${backupjob}

Test Delete Data
    [Tags]  backup  cassandra
    Delete From ${CASSANDRA_KEYSPACE} And Check
    Delete From ${GRANULAR_TEST_KEYSPACE} And Check

Test Restore Data
    [Tags]  backup  cassandra
    ${document}=  Set Variable  {"vault":"${backupjob}","dbs":["${CASSANDRA_KEYSPACE}","${GRANULAR_TEST_KEYSPACE}"]}
    Restore Data And Check   ${document}  ${ATTEMPTS_NUMBER}

Test Check All Restored Data
	[Tags]  backup  cassandra
    Check Data In Table  ${CASSANDRA_KEYSPACE}
    Check Data In Table  ${GRANULAR_TEST_KEYSPACE}  ${granular_backup_col1}  ${granular_backup_col2}

Test Recovery With System Keyspaces
    [Tags]  backup  cassandra
    ${keyspace}=  Set Variable  system_backup_${CASSANDRA_KEYSPACE}
    Create Data  ${keyspace}
    ${document}=  Set Variable  {"dbs":["${keyspace}","system","system_schema"]}
    ${backupjob}=  Backup Data And Check  ${document}  ${ATTEMPTS_NUMBER}
    Insert To ${keyspace} And Check
    ${document}=  Set Variable  {"vault":"${backupjob}","dbs":["${keyspace}","system"]}
    ${job}=  Restore Data And Check  ${document}  ${ATTEMPTS_NUMBER}  get_status_code=500
    Check Error In Jobstatus  ${job}  ${SYSTEM_KEYSPACE_ERROR_TEXT}  500
    [Teardown]  Run Keywords   DELETE KEYSPACE  ${keyspace}
    ...  AND  Delete Backup  ${backupjob}

Test Granular Restore From Full Backup
    [Tags]  backup  cassandra
    ${keyspace}=  Set Variable  granular_from_full_${CASSANDRA_KEYSPACE}
    Create Data  ${keyspace}
    ${backupjob}=  Backup Data And Check  ${None}  ${ATTEMPTS_NUMBER}
    Delete From ${keyspace} And Check
    ${document}=  Set Variable  {"vault":"${backupjob}","dbs":["${keyspace}"]}
    ${listbackups}=  GET On Session  backupsession  /listbackups
    Restore Data And Check   ${document}  ${ATTEMPTS_NUMBER}
    Check Data In Table  ${keyspace}
    [Teardown]  Run Keywords  DELETE KEYSPACE  ${keyspace}
    ...  AND  Delete Backup  ${backupjob}

Test Recovery With Empty Keyspace
    [Tags]  backup  cassandra
    ${empty_keyspace}=  Set Variable  empty_${CASSANDRA_KEYSPACE}
    Create Empty Keyspace  ${empty_keyspace}
    ${not_empty_keyspace}=  Set Variable  not_empty_${CASSANDRA_KEYSPACE}
    Create Data  ${not_empty_keyspace}

    ${document}=  Set Variable  {"dbs":["${empty_keyspace}", "${not_empty_keyspace}"]}
    ${backupjob}=  Backup Data And Check   ${document}  ${ATTEMPTS_NUMBER}
    Insert To ${not_empty_keyspace} And Check
    Delete Keyspace ${empty_keyspace} And Check

    ${document}=  Set Variable  {"vault":"${backupjob}","dbs":["${not_empty_keyspace}", "${empty_keyspace}"]}
    ${job}=  Restore Data And Check   ${document}  ${ATTEMPTS_NUMBER}
    ${resp}=  GET On Session  backupsession  /jobstatus/${job}
    Should Be Equal As Strings  ${resp.status_code}  200

    Check Data Not Exists In ${not_empty_keyspace}
    Check ${empty_keyspace} exists
    [Teardown]  Run Keywords  DELETE KEYSPACE  ${empty_keyspace}
    ...  AND  DELETE KEYSPACE  ${not_empty_keyspace}
    ...  AND  Delete Backup  ${backupjob}

Test Backup Eviction
    [Tags]  backup  cassandra
    ${keyspace}=  Set Variable  backup_eviction_${CASSANDRA_KEYSPACE}
    Create Data  ${keyspace}
    ${document}=  Set Variable  {"dbs":["${keyspace}"]}
    ${backupjob}=  Backup Data And Check   ${document}  ${ATTEMPTS_NUMBER}
    ${resp}=  GET On Session  backupsession  /listbackups/${backupjob}
    Should Be Equal As Strings  ${resp.status_code}  200
    Delete Backup  ${backupjob}
    ${listbackups}=  GET On Session  backupsession  /listbackups
    Should Be Equal As Strings  ${listbackups.status_code}  200
    Should Not Contain  ${listbackups.content}  ${backupjob}
    [Teardown]  DELETE KEYSPACE  ${keyspace}

Backup Saved On Pod After Restart
    [Tags]  ha  backup_ha  cassandra
    ${keyspace}=  Set Variable  ha_${CASSANDRA_KEYSPACE}
    Create Data  ${keyspace}
    ${document}=  Set Variable  {"dbs":["${keyspace}"]}
    ${backupjob}=  Backup Data And Check   ${document}  ${ATTEMPTS_NUMBER}
    ${daemon_pod_name}=  Get Pod Name For Deployment  cassandra-backup-daemon
    Check Backup Exists Using Baskup-daemon Pod  ${daemon_pod_name}  ${backupjob}
    Delete Pod By Pod Name  ${daemon_pod_name}  ${CASSANDRA_NAMESPACE}
    Sleep  15s
    ${daemon_pod_name}=  Get Pod Name For Deployment  cassandra-backup-daemon
    Wait Until Keyword Succeeds  2min  5s
    ...  Check Pod Status Is Running  ${daemon_pod_name}
    Check Backup Exists Using Baskup-daemon Pod  ${daemon_pod_name}  ${backupjob}
    [Teardown]  Run Keywords  DELETE KEYSPACE  ${keyspace}
    ...  AND  Delete Backup  ${backupjob}