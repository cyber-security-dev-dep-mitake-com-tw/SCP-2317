*** Settings ***
Library    Collections
Library    OperatingSystem
Library    ${CURDIR}/resources/ApiLibrary.py

*** Variables ***
${BASE_URL}       %{MAILGUARD_API_BASE_URL=http://127.0.0.1:8080}
${SAMPLES_DIR}    ${CURDIR}/../../samples

*** Test Cases ***
Health Endpoint Returns Ok
    ${text}=    Get Text    ${BASE_URL}/healthz
    Should Be Equal    ${text}    ok

Clean Fixture Stays Clean
    ${payload}=    Load Json File    ${SAMPLES_DIR}/clean.json
    ${verdict}=    Post Json    ${BASE_URL}/v1/email/scan    ${payload}
    ${status}=    Get From Dictionary    ${verdict}    status
    ${score}=    Get From Dictionary    ${verdict}    score
    Should Be Equal    ${status}    clean
    Should Be Equal As Integers    ${score}    0

Suspicious Fixture Is Flagged
    ${payload}=    Load Json File    ${SAMPLES_DIR}/suspicious.json
    ${verdict}=    Post Json    ${BASE_URL}/v1/email/scan    ${payload}
    ${status}=    Get From Dictionary    ${verdict}    status
    ${score}=    Get From Dictionary    ${verdict}    score
    Should Be Equal    ${status}    suspicious
    Should Be Equal As Integers    ${score}    25

Malicious Fixture Is Malicious
    ${payload}=    Load Json File    ${SAMPLES_DIR}/malicious.json
    ${verdict}=    Post Json    ${BASE_URL}/v1/email/scan    ${payload}
    ${status}=    Get From Dictionary    ${verdict}    status
    ${score}=    Get From Dictionary    ${verdict}    score
    Should Be Equal    ${status}    malicious
    Should Be Equal As Integers    ${score}    55

BEC Fixture Is Malicious
    ${payload}=    Load Json File    ${SAMPLES_DIR}/bec.json
    ${verdict}=    Post Json    ${BASE_URL}/v1/email/scan    ${payload}
    ${status}=    Get From Dictionary    ${verdict}    status
    Should Be Equal    ${status}    malicious

Quishing Fixture Is Flagged
    ${payload}=    Load Json File    ${SAMPLES_DIR}/quishing.json
    ${verdict}=    Post Json    ${BASE_URL}/v1/email/scan    ${payload}
    ${status}=    Get From Dictionary    ${verdict}    status
    Should Not Be Equal    ${status}    clean

Outbound DLP Blocks Card Number
    ${payload}=    Load Json File    ${SAMPLES_DIR}/dlp_outbound.json
    ${verdict}=    Post Json    ${BASE_URL}/v1/email/scan    ${payload}
    ${status}=    Get From Dictionary    ${verdict}    status
    Should Be Equal    ${status}    malicious

Missing Message Id Returns Structured Error
    ${payload}=    Evaluate    {'sender': 'a@b.test', 'subject': 'x'}
    ${error}=    Post Json Expect Error    ${BASE_URL}/v1/email/scan    ${payload}    400
    Dictionary Should Contain Key    ${error}    error

Sandbox Stub Never Executes
    ${payload}=    Evaluate    {'hash': 'abc', 'objectRef': 'obj://abc', 'policy': 'default'}
    ${sub}=    Post Json Accepted    ${BASE_URL}/v1/sandbox/submit    ${payload}
    ${job_id}=    Get From Dictionary    ${sub}    job_id
    ${report}=    Get Json    ${BASE_URL}/v1/sandbox/report/${job_id}
    ${status}=    Get From Dictionary    ${report}    status
    Should Be Equal    ${status}    not_executed
