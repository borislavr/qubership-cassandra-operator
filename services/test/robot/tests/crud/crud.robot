*** Variables ***

*** Settings ***
Library  String
Resource  ../shared/keywords.robot
Suite Setup  Preparation
Suite Teardown  Cleanup

*** Keywords ***
Preparation
    Prepare Shared
    ${CASSANDRA_KEYSPACE}=  Generate Random String  10  [LOWER]
    Set Suite Variable  ${CASSANDRA_KEYSPACE}

Cleanup
    DELETE KEYSPACE  ${CASSANDRA_KEYSPACE}

*** Test Cases ***
Test Cassandra connection
    [Tags]  smoke  cassandra
	Connect To Cassandra  ${TLS_ENABLED}

Test Create Keyspace
    [Tags]  smoke  cassandra
    Create Keyspace    ${CASSANDRA_KEYSPACE}  ${DC_NAME}  ${TEST_KEYSPACES_REPLICATION_FACTOR}

Test Create Table
    [Tags]  smoke  cassandra
    Create Table    ${CASSANDRA_KEYSPACE}

Test Insert Data
    [Tags]  smoke  cassandra
    Insert To ${CASSANDRA_KEYSPACE} And Check

Test Delete Data
    [Tags]  smoke  cassandra
    Delete From ${CASSANDRA_KEYSPACE} And Check