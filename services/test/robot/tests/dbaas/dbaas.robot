*** Variables ***
${DBAAS_HOST}                                     %{DBAAS_HOST}
${DB_NAME}                                        dbaas_test
${ATTEMPTS_NUMBER}                                %{ATTEMPTS_NUMBER=100}

*** Settings ***
Library  json
Library  String
Library	 Collections
Library	 RequestsLibrary
Resource  ../shared/keywords.robot
Resource  dbaas_shared.robot
Suite Setup  Preparation
Suite Teardown  Cleanup

*** Keywords ***
Preparation
    Prepare Shared
    Preparation dbaas shared
    Check Enabled Multiple Users

Cleanup
	DELETE KEYSPACE  ${resultkeyspace}
    DELETE KEYSPACE  ${CASSANDRA_KEYSPACE}
    DELETE KEYSPACE  ${GRANULAR_TEST_KEYSPACE}

Backup Data And Check
    [Arguments]  ${document}  ${attempts}
    ${response}=  Post Request With ${document} Data To /api/${dbaas_api_version}/dbaas/adapter/cassandra/backups/collect
    Should Be Equal As Strings  ${response.status_code}  202
    ${resultjson}=    Evaluate     json.loads("""${response.content}""")    json
    ${backupId}=  Set Variable  ${resultjson['trackId']}
    ${response}=  Wait For /api/${dbaas_api_version}/dbaas/adapter/cassandra/backups/track/backup/${backupId} Job Completion With ${attempts} Attempts
    Should Be Equal As Strings  ${response.status_code}  200
    RETURN  ${backupId}

Restore Data And Check
    [Arguments]  ${document}  ${backupId}  ${attempts}
    ${response}=  Post Request With ${document} Data To /api/${dbaas_api_version}/dbaas/adapter/cassandra/backups/${backupId}/restore
    Should Be Equal As Strings  ${response.status_code}  202
    ${resultjson}=    Evaluate     json.loads("""${response.content}""")    json
    ${restoreId}=  Set Variable  ${resultjson['trackId']}
    ${response}=  Wait For /api/${dbaas_api_version}/dbaas/adapter/cassandra/backups/track/restore/${restoreId} Job Completion With ${attempts} Attempts
    Should Be Equal As Strings  ${response.status_code}  200

Restore Data With Regenerate Names
    [Arguments]  ${document}  ${backupId}  ${attempts}
    ${response}=  Post Request With ${document} Data To /api/${dbaas_api_version}/dbaas/adapter/cassandra/backups/${backupId}/restore?regenerateNames=true
    Should Be Equal As Strings  ${response.status_code}  202
    ${resultjson}=    Evaluate     json.loads("""${response.content}""")    json
    ${restoreId}=  Set Variable  ${resultjson['trackId']}
    ${response}=  Wait For /api/${dbaas_api_version}/dbaas/adapter/cassandra/backups/track/restore/${restoreId} Job Completion With ${attempts} Attempts
    Should Be Equal As Strings  ${response.status_code}  200
    RETURN  ${resultjson}

Check Roles Existence In Response
    [Arguments]  ${resp}
    ${resp_con_properties}=  Get From Dictionary  ${resp.json()}  connectionProperties
    ${length} =	Get Length	${resp_con_properties}
    Should Be Equal As Integers	 ${length}  4
    Should Contain  str(${resp_con_properties})  'role': 'admin'
    Should Contain  str(${resp_con_properties})  'role': 'streaming'
    Should Contain  str(${resp_con_properties})  'role': 'rw'
    Should Contain  str(${resp_con_properties})  'role': 'ro'

Check Permissions For Role
    [Arguments]  ${db_users}  ${role_name}  ${expected_permissions}
    ${role}=  Get From Dictionary  ${db_users}  ${role_name}
    ${permissions}=  get_permission_for_role  ${role}
    Validate Permissions  ${expected_permissions}  ${permissions}
    
Validate Permissions
    [Arguments]  ${expected_permissions}  ${actual_permissions}
    Lists Should Be Equal  ${expected_permissions}  ${actual_permissions}  Actual Permissions: ${actual_permissions}, Expected Permissions: ${expected_permissions}

Check Enabled Multiple Users
    ${env_variables}=  Create List  MULTI_USERS_ENABLED
    ${envs}=  Get Environment Variables For Deployment Entity Container  dbaas-cassandra-adapter  ${CASSANDRA_NAMESPACE}  dbaas-cassandra-adapter  ${env_variables}
    ${multiple_users_enabled}=  Get From Dictionary  ${envs}  MULTI_USERS_ENABLED
    Set Suite Variable  ${multiple_users_enabled}

Create Keyspace And Return Users Names
    ${data}=  Set Variable  {"metadata":{"test-meta":"meta-val"}, "dbName": "${DB_NAME}", "namePrefix" :"", "password": "pass", "username": "dbaas_test"}
    ${resp}=  POST On Session  dbaassession  /api/${dbaas_api_version}/dbaas/adapter/cassandra/databases  data=${data}
    Should Be Equal As Strings  ${resp.status_code}  201
    Dictionary Should Contain Key  ${resp.json()}  name
    ${resp_name}=  Get From Dictionary  ${resp.json()}  name
    Should Be Equal  ${DB_NAME}  ${resp_name}
    Check ${DB_NAME} exists
    Check Roles Existence In Response  ${resp}
    ${resp_con_properties}=  Get From Dictionary  ${resp.json()}  connectionProperties
    ${db_users}=  Get Multiple Users Name  ${resp_con_properties}
    RETURN  ${db_users}

Check Users Permissions
    [Arguments]  ${cassandra_version}  ${db_users}
    ${ADMIN_PERMISSIONS} =    Get Expected Permissions  ${cassandra_version}
    Check Permissions For Role  ${db_users}  admin  ${ADMIN_PERMISSIONS}
    ${ro_permissions} =	Create List  SELECT
    Check Permissions For Role  ${db_users}  ro  ${ro_permissions}
    ${rw_permissions} =	Create List  SELECT  MODIFY
    Check Permissions For Role  ${db_users}  rw  ${rw_permissions}
    Check Permissions For Role  ${db_users}  streaming  ${ADMIN_PERMISSIONS}

Get Expected Permissions
    [Arguments]  ${cassandra_version}
    Log  Determining expected permissions for Cassandra version: "${cassandra_version}"
    ${contains} =    Evaluate    "${cassandra_version}".startswith("5.")
    ${permissions}=  Run Keyword If    ${contains}
    ...    Create List    CREATE    ALTER    DROP    SELECT    MODIFY    AUTHORIZE    UNMASK    SELECT_MASKED
    ...    ELSE
    ...    Create List    CREATE    ALTER    DROP    SELECT    MODIFY    AUTHORIZE
    RETURN  ${permissions}

Delete Dbaas Users
    [Arguments]  ${db_users}
    ${user_name}=  Get From Dictionary  ${db_users}  admin
    Delete User  ${user_name}
    ${user_name}=  Get From Dictionary  ${db_users}  ro
    Delete User  ${user_name}
    ${user_name}=  Get From Dictionary  ${db_users}  rw
    Delete User  ${user_name}
    ${user_name}=  Get From Dictionary  ${db_users}  streaming
    Delete User  ${user_name}

*** Test Cases ***
Test Wrong Credentials
    [Tags]  dbaas  cassandra
    ${wronguser}=  Generate Random String  10
    ${wrongpass}=  Generate Random String  10
    @{connection_settings}=  Prepare Configuration For Dbaas Connection
    Create Session    wrongcredssession    ${connection_settings[0]}://${wronguser}:${wrongpass}@${DBAAS_HOST}:${connection_settings[1]}  verify=${connection_settings[2]}
    ${response}=  GET On Session  wrongcredssession  /api/${dbaas_api_version}/dbaas/adapter/cassandra/databases  expected_status=401
    Should Be Equal As Strings  ${response.status_code}  401

Test Create Keyspace
    [Tags]  dbaas  cassandra
    ${document}=  Set Variable  {"dbName":"${CASSANDRA_KEYSPACE}","namePrefix":"${namePrefix}","metadata":{"${meta_string}":"${meta_value}"}}
    ${response}=  Post Request With ${document} Data To /api/${dbaas_api_version}/dbaas/adapter/cassandra/databases
    Should Be Equal As Strings  ${response.status_code}  201
    Should Contain  b"${response.content}"  ${resultkeyspace}

    ${resultjson}=    Evaluate     json.loads("""${response.content}""")    json
    ${users_amount}=  Get length    ${resultjson['connectionProperties']}

    Run Keyword If  "${multiple_users_enabled}" == "true" and "${dbaas_api_version}" != "v1"
    ...  Should Be Equal As Integers  ${users_amount}  ${4}  Multiple Users Is Enabled, the number of created users is not equal to 4
    ...  ELSE
    ...  Should Be Equal As Integers  ${users_amount}  ${1}  Multiple Users Is Disabled, the number of created users is not equal to 1

Test Check Keyspace List
    [Tags]  dbaas  cassandra
    ${response}=  Get Request To /api/${dbaas_api_version}/dbaas/adapter/cassandra/databases
    Should Be Equal As Strings  ${response.status_code}  200
    Should Contain  b"${response.content}"  ${resultkeyspace}

Test Check Meta In Database
	[Tags]  dbaas  cassandra
    Check ${meta_string} In ${resultkeyspace} metadata
    Check ${meta_value} In ${resultkeyspace} metadata

Test Update Meta In Database
    [Tags]  dbaas  cassandra
    ${updmetacol}=  Generate Random String  10
    ${updmetaval}=  Generate Random String  10

    ${document}=  Set Variable  {"${updmetacol}": "${updmetaval}"}
    ${response}=  Put Request With ${document} Data To /api/${dbaas_api_version}/dbaas/adapter/cassandra/databases/${resultkeyspace}/metadata
    Log  ${response.content}
    Should Be Equal As Strings  ${response.status_code}  200

    Check ${updmetacol} In ${resultkeyspace} metadata
    Check ${updmetaval} In ${resultkeyspace} metadata

Test Create Users To Database
    [Tags]  dbaas  cassandra
    ${document}=  Set Variable  {"dbName": "${resultkeyspace}","password":null}
    ${response}=  Put Request With ${document} Data To /api/${dbaas_api_version}/dbaas/adapter/cassandra/users
    Log  ${response.content}
    Should Be Equal As Strings  ${response.status_code}  201
    ${resultjson}=    Evaluate     json.loads("""${response.content}""")    json

    ${generateduser}=  Set Variable  ${resultjson['connectionProperties']['username']}
    ${generatedpass}=  Set Variable  ${resultjson['connectionProperties']['password']}

    Log  ${generateduser}
    Log  ${generatedpass}

    ${usertocreate}=  Catenate    SEPARATOR=  test   ${generateduser}
    ${passtocreate}=  Catenate    SEPARATOR=  test   ${generatedpass}

    ${document}=  Set Variable  {"dbName": "${resultkeyspace}","password":"${passtocreate}"}
    ${response}=  Put Request With ${document} Data To /api/${dbaas_api_version}/dbaas/adapter/cassandra/users/${usertocreate}
    Log  ${response.content}
    Should Be Equal As Strings  ${response.status_code}  201
    ${resultjson}=    Evaluate     json.loads("""${response.content}""")    json

    ${predefineduser}=  Set Variable  ${resultjson['connectionProperties']['username']}
    ${predefinedpass}=  Set Variable  ${resultjson['connectionProperties']['password']}

    Log  ${predefineduser}
    Log  ${predefinedpass}

    Should Be Equal As Strings  ${predefineduser}  ${usertocreate}
    Should Be Equal As Strings  ${predefinedpass}  ${passtocreate}

Test Drop Database
	[Tags]  dbaas  cassandra
	${document}=  Set Variable  [{"kind": "database","name": "${resultkeyspace}"}]
	${response}=  Post Request With ${document} Data To /api/${dbaas_api_version}/dbaas/adapter/cassandra/resources/bulk-drop
	Should Be Equal As Strings  ${response.status_code}  200

    ${response}=  Get Request To /api/${dbaas_api_version}/dbaas/adapter/cassandra/databases
    Should Be Equal As Strings  ${response.status_code}  200
    Should Not Contain  b"${response.content}"  ${resultkeyspace}


Test Create Granular Data Directly
    [Tags]  dbaas_backup  cassandra
    Create Keyspace    ${GRANULAR_TEST_KEYSPACE}  ${DC_NAME}  ${TEST_KEYSPACES_REPLICATION_FACTOR}
    Create Table    ${GRANULAR_TEST_KEYSPACE}
    Insert To ${GRANULAR_TEST_KEYSPACE} And Check
    ${granular_backup_col1}=  Set Variable  ${col1}
    ${granular_backup_col2}=  Set Variable  ${col2}
    Set Suite Variable  ${granular_backup_col1}
    Set Suite Variable  ${granular_backup_col2}

Test Granular Backup
    [Tags]  dbaas_backup  cassandra
    ${document}=  Set Variable  ["${GRANULAR_TEST_KEYSPACE}"]
    ${granularBackupId}=  Backup Data And Check  ${document}  ${ATTEMPTS_NUMBER}
    Set Suite Variable  ${granularBackupId}

Test Delete Granular Data Directly
    [Tags]  dbaas_backup  cassandra
    Delete From ${GRANULAR_TEST_KEYSPACE} And Check

Test Granular Restore
    [Tags]  dbaas_backup  cassandra
    ${document}=  Set Variable  ["${GRANULAR_TEST_KEYSPACE}"]
    Restore Data And Check  ${document}  ${granularBackupId}  ${ATTEMPTS_NUMBER}

Test Check Restored Granular Data Directly
    [Tags]  dbaas_backup  cassandra
    Check Data In Table  ${GRANULAR_TEST_KEYSPACE}

Test Create Data Directly
    [Tags]  dbaas_backup  cassandra
    Create Keyspace    ${CASSANDRA_KEYSPACE}  ${DC_NAME}  ${TEST_KEYSPACES_REPLICATION_FACTOR}
    Create Table    ${CASSANDRA_KEYSPACE}
    Insert To ${CASSANDRA_KEYSPACE} And Check

Test Full Backup
    [Tags]  dbaas_backup  cassandra
    ${document}=  Set Variable  []
    ${fullBackupId}=  Backup Data And Check  ${document}  ${ATTEMPTS_NUMBER}
    Set Suite Variable  ${fullBackupId}

Test Delete Data Directly
    [Tags]  dbaas_backup  cassandra
    Delete From ${CASSANDRA_KEYSPACE} And Check
    Delete From ${GRANULAR_TEST_KEYSPACE} And Check

Test Full Restore
    [Tags]  dbaas_backup  cassandra
    ${document}=  Set Variable  ["${CASSANDRA_KEYSPACE}","${GRANULAR_TEST_KEYSPACE}"]
    Restore Data And Check  ${document}  ${fullBackupId}  ${ATTEMPTS_NUMBER}

Test Check All Restored Data Directly
    [Tags]  dbaas_backup  cassandra
    Check Data In Table  ${CASSANDRA_KEYSPACE}
    Check Data In Table  ${GRANULAR_TEST_KEYSPACE}  ${granular_backup_col1}  ${granular_backup_col2}

Test Recovery With RegenerateNames
    [Tags]  dbaas_backup  cassandra
    ${REGENERATENAMES_BACKUP_KEYSPACE}=  Set Variable  regenerate_names_${CASSANDRA_KEYSPACE}
    Create Data  ${REGENERATENAMES_BACKUP_KEYSPACE}
    ${document}=  Set Variable  ["${REGENERATENAMES_BACKUP_KEYSPACE}"]
    ${granularBackupId}=  Backup Data And Check  ${document}  ${ATTEMPTS_NUMBER}
    Check Data In Table  ${REGENERATENAMES_BACKUP_KEYSPACE}
    Delete From ${REGENERATENAMES_BACKUP_KEYSPACE} And Check
    ${resultjson}=  Restore Data With Regenerate Names  ${document}  ${granularBackupId}  ${ATTEMPTS_NUMBER}
    ${dict}=  Set Variable  ${resultjson['changedNameDb']}
    ${new_name}=  Set Variable  ${dict['${REGENERATENAMES_BACKUP_KEYSPACE}']}
    Should Be True  """${REGENERATENAMES_BACKUP_KEYSPACE}_clone""" in """${new_name}"""
    Check Data In Table  ${new_name}
    [Teardown]  Run Keywords  DELETE KEYSPACE  ${REGENERATENAMES_BACKUP_KEYSPACE}
     ...  AND  DELETE KEYSPACE  ${new_name}

Test Multiple Users Creating
    [Tags]  dbaas  dbaas_multiple_users  cassandra
    Skip If  "${dbaas_api_version}" == "v1"  API version v1, not possible to check case!
    Skip If  "${multiple_users_enabled}" == "false"  MULTI_USERS_ENABLED = False, not possible to check case!
    ${cassandra_version}=      Fetch Cassandra Version
    ${db_users}=  Create Keyspace And Return Users Names
    Check Users Permissions  ${cassandra_version}  ${db_users}
    [Teardown]  Run Keywords  DELETE KEYSPACE  ${DB_NAME}
     ...  AND  Delete Dbaas Users  ${db_users}

Test Users Backup And Restore
    [Tags]  dbaas  dbaas_multiple_users  cassandra  dbaas_backup
    Skip If  "${dbaas_api_version}" == "v1"  API version v1, not possible to check case!
    Skip If  "${multiple_users_enabled}" == "false"  MULTI_USERS_ENABLED = False, not possible to check case!
    ${cassandra_version}=      Fetch Cassandra Version
    ${db_users}=  Create Keyspace And Return Users Names
    Create Data  ${DB_NAME}
    ${all_users}=  Get All Users
    ${db_admin_user_name}=  Get From Dictionary  ${db_users}  admin
    List Should Contain Value  ${all_users}  ${db_admin_user_name}
    Check Users Permissions  ${cassandra_version}  ${db_users}
    ${document}=  Set Variable  ["${DB_NAME}"]
    ${granularBackupId}=  Backup Data And Check  ${document}  ${ATTEMPTS_NUMBER}
    Delete User  ${db_admin_user_name}
    ${db_rw_user_name}=  Get From Dictionary  ${db_users}  rw
    Revoke Permission From User  MODIFY  KEYSPACE ${DB_NAME}  ${db_rw_user_name}
    ${all_users}=  Get All Users
    List Should Not Contain Value  ${all_users}  ${db_admin_user_name}
    Restore Data And Check  ${document}  ${granularBackupId}  ${ATTEMPTS_NUMBER}
    ${all_users}=  Get All Users
    List Should Contain Value  ${all_users}  ${db_admin_user_name}
    Check Users Permissions  ${cassandra_version}  ${db_users}
    [Teardown]  Run Keywords  DELETE KEYSPACE  ${DB_NAME}
     ...  AND  Delete Dbaas Users  ${db_users}
