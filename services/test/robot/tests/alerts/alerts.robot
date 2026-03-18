*** Variables ***
${CASSANDRA_STATEFUL_SET_NAME}           cassandra0
${CASSANDRA_POD_NAME}                    cassandra1-0
${PODS_COUNT_IS_LOVER_THAN_EXPECTED}     Cassandra pods count is lower than expected
${CASSANDRA_POD_IS_NOT_RUNNING}          Cassandra pod is not Running
${ALERT_RETRY_TIME}                      5min
${ALERT_RETRY_INTERVAL}                  1s
${SLEEP_TIME}                            30s

*** Settings ***
Library  MonitoringLibrary  host=%{PROMETHEUS_URL}
Resource  ../shared/keywords.robot
Library  PlatformLibrary  managed_by_operator=true

*** Keywords ***
Check That Prometheus Alert Is Active
    [Arguments]  ${alert_name}
    ${status}=  Get Alert Status  ${alert_name}  ${CASSANDRA_NAMESPACE}
    Should Be Equal As Strings  ${status}  pending

Check That Prometheus Alert Is Inactive
    [Arguments]  ${alert_name}
    ${status}=  Get Alert Status  ${alert_name}  ${CASSANDRA_NAMESPACE}
    Should Be Equal As Strings  ${status}  inactive

Delete Pod And Check Alert
    [Arguments]  ${pod_name}  ${alert}
    Delete Pod By Pod Name  ${pod_name}  ${CASSANDRA_NAMESPACE}  60
    Check That Prometheus Alert Is Active  ${alert}

Check Alerts Are Inactive
    Wait Until Keyword Succeeds  ${ALERT_RETRY_TIME}  ${ALERT_RETRY_INTERVAL}  Run Keywords
    ...  Check That Prometheus Alert Is Inactive  ${PODS_COUNT_IS_LOVER_THAN_EXPECTED}
    ...  AND  Check That Prometheus Alert Is Inactive  ${CASSANDRA_POD_IS_NOT_RUNNING}

Scale Up Stateful Set And Check
    Set Replicas For Stateful Set  ${CASSANDRA_STATEFUL_SET_NAME}  ${CASSANDRA_NAMESPACE}  1
    Wait Until Keyword Succeeds  ${ALERT_RETRY_TIME}  ${ALERT_RETRY_INTERVAL}
    ...  Check Pod Status Is Running  ${CASSANDRA_STATEFUL_SET_NAME}-0

*** Test Cases ***
Pods Count Is Lower Than Expected Alert
    [Tags]  alerts  cassandra
    Check That Prometheus Alert Is Inactive  ${PODS_COUNT_IS_LOVER_THAN_EXPECTED}
    Set Replicas For Stateful Set  ${CASSANDRA_STATEFUL_SET_NAME}  ${CASSANDRA_NAMESPACE}  0
    Wait Until Keyword Succeeds  ${ALERT_RETRY_TIME}  ${ALERT_RETRY_INTERVAL}
    ...  Check That Prometheus Alert Is Active  ${PODS_COUNT_IS_LOVER_THAN_EXPECTED}
    Set Replicas For Stateful Set  ${CASSANDRA_STATEFUL_SET_NAME}  ${CASSANDRA_NAMESPACE}  1
    Check Alerts Are Inactive
    [Teardown]  Scale Up Stateful Set And Check

Cassandra Pod Is Not Running Alert
    [Tags]  alerts  cassandra
    Check That Prometheus Alert Is Inactive  ${CASSANDRA_POD_IS_NOT_RUNNING}
    Wait Until Keyword Succeeds  ${ALERT_RETRY_TIME}  ${ALERT_RETRY_INTERVAL}
    ...  Delete Pod And Check Alert  ${CASSANDRA_POD_NAME}  ${CASSANDRA_POD_IS_NOT_RUNNING}
    Check Alerts Are Inactive

