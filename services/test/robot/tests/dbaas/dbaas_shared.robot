*** Variables ***
${DBAAS_HOST}                                     %{DBAAS_HOST}

*** Settings ***
Library  String
Library	 Collections
Library	 RequestsLibrary
Library  OperatingSystem

*** Keywords ***
Preparation dbaas shared
    Load Secrets

    &{headers}=  Create Dictionary  Content-Type=application/json  Accept=application/json
    Set Suite Variable  ${headers}

    ${CASSANDRA_KEYSPACE}=  Generate Random String  10  [LOWER]
    Set Suite Variable  ${CASSANDRA_KEYSPACE}

    ${GRANULAR_TEST_KEYSPACE}=  Set Variable  granular_${CASSANDRA_KEYSPACE}
    Set Suite Variable  ${GRANULAR_TEST_KEYSPACE}

    ${namePrefix}=  Catenate    SEPARATOR=   ${CASSANDRA_KEYSPACE}
    Set Suite Variable  ${namePrefix}

    ${resultkeyspace}=  Catenate    SEPARATOR=   ${namePrefix}   ${CASSANDRA_KEYSPACE}
    Set Suite Variable  ${resultkeyspace}

    ${meta_string}=  Generate Random String  10
    ${meta_value}=  Generate Random String  10

    Set Suite Variable  ${meta_string}
    Set Suite Variable  ${meta_value}

    @{connection_settings}=  Prepare Configuration For Dbaas Connection
    Create Session    dbaassession    ${connection_settings[0]}://${DBAAS_ADAPTER_USERNAME}:${DBAAS_ADAPTER_PASSWORD}@${DBAAS_HOST}:${connection_settings[1]}    verify=${connection_settings[2]}

    Import Library  ${CURDIR}/../lib/CassandraLibrary.py  ${CASSANDRA_HOST}  ${CASSANDRA_USERNAME}  ${CASSANDRA_PASSWORD}  ${WAIT_TIMEOUT}

    ${dbaas_api_version}=    Get Dbaas Aggregator version
    Set Suite Variable  ${dbaas_api_version}

Get Dbaas Aggregator version 
    ${env_dbaas_aggregator_host}=  Create List  DBAAS_AGGREGATOR_REGISTRATION_ADDRESS
    ${dbaas_aggregator_host}=  Get Environment Variables For Deployment Entity Container  dbaas-cassandra-adapter  ${CASSANDRA_NAMESPACE}  dbaas-cassandra-adapter  ${env_dbaas_aggregator_host}
    Create Session    dbaas_aggregator   ${dbaas_aggregator_host["DBAAS_AGGREGATOR_REGISTRATION_ADDRESS"]}
    ${resp}=    Run Keyword And Ignore Error    Get On Session    dbaas_aggregator    /api-version

    IF    '${resp[0]}' == 'PASS' 
        IF    '${resp[1].status_code}' == '200'
            ${version}=    Evaluate     json.loads("""${resp[1].content}""")    json
            IF    3 in $version["supportedMajors"]
                ${apiVersion}=    Set Variable    v2
            ELSE
                ${apiVersion}=    Set Variable    v1
            END
        END
    ELSE
        ${apiVersion}=    Set Variable    v2
    END

    RETURN    ${apiVersion}
 
Prepare Configuration For Dbaas Connection
    ${verify}=    Get Environment Variable    name=TLS_ROOTCERT    default=False
    ${dbaas_tls}=    Get Environment Variable    name=TLS_ENABLED    default=False
    ${port}=    Get Environment Variable    name=PORT    default=8080
    ${env_dbaas_aggregator_host}=  Create List  DBAAS_AGGREGATOR_REGISTRATION_ADDRESS
    ${dbaas_aggregator_host}=  Get Environment Variables For Deployment Entity Container  dbaas-cassandra-adapter  ${CASSANDRA_NAMESPACE}  dbaas-cassandra-adapter  ${env_dbaas_aggregator_host}
    ${https_aggregator_enabled}=  Evaluate  "https" in "${dbaas_aggregator_host}"
    Set Suite Variable  ${https_aggregator_enabled}
    @{connection_settings}=  Run Keyword If  '${https_aggregator_enabled}' == '${True}' and '${dbaas_tls}' == 'true'
    ...  Set Variable  https  ${port}  ${verify}
    ...  ELSE
    ...  Set Variable  http  8080  False
    log to console  PORT, PROTOCOL and VERIFY for DBAAS Connection: ${connection_settings[0]} ${connection_settings[1]} ${connection_settings[2]}
    RETURN  @{connection_settings}


Get Request To ${point}
    ${heads}=  Set Variable  ${headers}

    ${resp}=  GET On Session  dbaassession  ${point}

    Log  ${resp}
    Log  ${resp.status_code}
    Log  ${resp.content}

    RETURN  ${resp}

Post Request With ${doc} Data To ${point}
    ${heads}=  Set Variable  ${headers}

    Log  dbaassession

    ${resp}=  POST On Session  dbaassession  ${point}  data=${doc}  headers=${heads}

    Log  ${resp}
    Log  ${resp.status_code}
    Log  ${resp.content}

    RETURN  ${resp}

Put Request With ${doc} Data To ${point}
    ${heads}=  Set Variable  ${headers}

    ${resp}=  PUT On Session  dbaassession  ${point}  data=${doc}  headers=${heads}

    Log  ${resp}
    Log  ${resp.status_code}
    Log  ${resp.content}

    RETURN  ${resp}

Check ${col} In ${keyspace} ${table}
    Log  ${col}
    Log  ${table}
    ${result}=  Select From Table  ${keyspace}  ${table}
    Log  ${result}
    Should Contain  ${result[1]}  ${col}

Wait For ${job} Job Completion With ${attempts} Attempts
    ${last_index}    Evaluate    ${attempts} - 1
    FOR    ${CheckStatus}    IN RANGE    ${attempts}
        ${resp}=  GET On Session  dbaassession  ${job}
        Log  ${resp.content}
        Exit For Loop If    '${resp.status_code}'=='400'
        Exit For Loop If    '${resp.status_code}'=='401'
        Exit For Loop If    '${resp.status_code}'=='403'
        Exit For Loop If    '${resp.status_code}'=='404'
        Exit For Loop If    '${resp.status_code}'=='500'
        ${resultjson}=    Evaluate     json.loads("""${resp.content}""")    json
        Exit For Loop If    '${resultjson['status']}'=='SUCCESS'
        Exit For Loop If    '${resultjson['status']}'=='FAIL'
        Sleep  5s
        Run Keyword If  ${CheckStatus} == ${last_index}  Fail  Job hasn't finished successfully in ${attempts} attempts, result: ${resultjson}. Increase attempts!
    END
    Log  ${resp}
    Log  ${resp.content}
    RETURN  ${resp}
