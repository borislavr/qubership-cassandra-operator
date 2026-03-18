*** Variables ***
${CASSANDRA_STATEFUL_SET_NAME}           cassandra0
${ALERT_RETRY_TIME}                      5min
${ALERT_RETRY_INTERVAL}                  1s
${SLEEP_TIME}                            30s

*** Settings ***
Resource  ../shared/keywords.robot
Library  PlatformLibrary  managed_by_operator=true
Suite Setup  Preparation
Suite Teardown  Cleanup

*** Keywords ***
Preparation
    Prepare Shared
    ${CASSANDRA_KEYSPACE}=  Generate Random String  10  [LOWER]
    Set Suite Variable  ${CASSANDRA_KEYSPACE}

Cleanup
    DELETE KEYSPACE  ${CASSANDRA_KEYSPACE}
    DELETE KEYSPACE  ${CASSANDRA_KEYSPACE}2

*** Test Cases ***
Test HA Case
    [Tags]  ha  cassandra
#    Pass Execution If  ${TEST_KEYSPACES_REPLICATION_FACTOR} < 3  Replicas < 3, not possible to check case!
    Skip If  ${TEST_KEYSPACES_REPLICATION_FACTOR} < 3  Replicas < 3, not possible to check case!
    Create Data  ${CASSANDRA_KEYSPACE}
    Set Replicas For Stateful Set  ${CASSANDRA_STATEFUL_SET_NAME}  ${CASSANDRA_NAMESPACE}  0
    Check Service Of Stateful Sets Is Scaled  ${CASSANDRA_STATEFUL_SET_NAME}  ${CASSANDRA_NAMESPACE}  down
    Insert To ${CASSANDRA_KEYSPACE} And Check
    Set Replicas For Stateful Set  ${CASSANDRA_STATEFUL_SET_NAME}  ${CASSANDRA_NAMESPACE}  1
    Check Service Of Stateful Sets Is Scaled  ${CASSANDRA_STATEFUL_SET_NAME}  ${CASSANDRA_NAMESPACE}
    Sleep  10s
    ${result}=  Select From Table  ${CASSANDRA_KEYSPACE}
    Log To Console  RESULT: ${result}
    ${cnt}=  Get length  ${result}
    Should Be Equal As Numbers  ${cnt}  4
    Create Data  ${CASSANDRA_KEYSPACE}2