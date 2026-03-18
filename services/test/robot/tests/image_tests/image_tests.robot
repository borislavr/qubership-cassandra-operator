*** Settings ***
Library    String
Library    Collections
Resource   ../shared/keywords.robot

*** Variables ***
${CASSANDRA_NAMESPACE}           %{NAMESPACE}
${CONFIG_NAME}                   %{CONFIG_NAME}
${SUPPLEMENTARY_CONFIG_NAME}     %{SUPPLEMENTARY_CONFIG_NAME}

*** Keywords ***
Get Image Tag
    [Arguments]  ${image}
    ${parts}=  Split String  ${image}  :
    ${length}=  Get Length  ${parts}
    Run Keyword If  ${length} > 1  Return From Keyword  ${parts[2]}  ELSE  Fail  Image has no tag: ${image}

Compare Images From Resources With Dd
    [Arguments]  ${dd_images}
    ${stripped_resources}=  Strip String  ${dd_images}  characters=,  mode=right
    @{list_resources} =  Split String  ${stripped_resources}  ,
    FOR  ${resource}  IN  @{list_resources}
        ${type}  ${name}  ${container_name}  ${image}=  Split String  ${resource}
        ${resource_image}=  Get Resource Image  ${type}  ${name}  ${CASSANDRA_NAMESPACE}  ${container_name}

        ${expected_tag}=  Get Image Tag  ${image}
        ${actual_tag}=    Get Image Tag  ${resource_image}

        Log To Console  \n[COMPARE] ${resource}: Expected tag = ${expected_tag}, Actual tag = ${actual_tag}
        Run Keyword And Continue On Failure  Should Be Equal   ${actual_tag}   ${expected_tag}
    END

*** Test Cases ***
Test Hardcoded Images For Cassandra
    [Tags]  check_cassandra_images  smoke  cassandra
    ${dd_images}=  Get Dd Images From Config Map  ${CONFIG_NAME}  ${CASSANDRA_NAMESPACE}
    Skip If  '${dd_images}' == '${None}'  There is no deployDescriptor, not possible to check case!
    Compare Images From Resources With Dd  ${dd_images}

Test Hardcoded Images For Supplementary Services
    [Tags]  check_cassandra_images  smoke  cassandra
    ${dd_images}=  Get Dd Images From Config Map  ${SUPPLEMENTARY_CONFIG_NAME}  ${CASSANDRA_NAMESPACE}
    Skip If  '${dd_images}' == '${None}'  There is no deployDescriptor, not possible to check case!
    Compare Images From Resources With Dd  ${dd_images}
