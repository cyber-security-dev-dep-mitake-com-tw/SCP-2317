*** Settings ***
Library    Collections
Library    ${CURDIR}/resources/ApiLibrary.py

*** Variables ***
${BASE_URL}    %{MAILGUARD_API_BASE_URL=http://127.0.0.1:8080}

*** Test Cases ***
Health Endpoint Returns Ok
    ${text}=    Get Text    ${BASE_URL}/healthz
    Should Be Equal    ${text}    ok

Clean Message Stays Clean
    ${payload}=    Evaluate    {'message_id': 'robot-clean', 'sender': 'sender@example.test', 'subject': 'Quarterly update', 'body_preview': 'Hello', 'urls': ['https://example.com'], 'attachments': []}
    ${verdict}=    Post Json    ${BASE_URL}/v1/email/scan    ${payload}
    ${status}=    Get From Dictionary    ${verdict}    status
    ${score}=    Get From Dictionary    ${verdict}    score
    Should Be Equal    ${status}    clean
    Should Be Equal As Integers    ${score}    0

Suspicious Url Is Flagged
    ${payload}=    Evaluate    {'message_id': 'robot-url', 'sender': 'sender@example.test', 'subject': 'Suspicious link', 'body_preview': 'Click carefully', 'urls': ['http://xn--demo.test'], 'attachments': []}
    ${verdict}=    Post Json    ${BASE_URL}/v1/email/scan    ${payload}
    ${status}=    Get From Dictionary    ${verdict}    status
    ${score}=    Get From Dictionary    ${verdict}    score
    Should Be Equal    ${status}    suspicious
    Should Be Equal As Integers    ${score}    25

Executable Attachment Is Malicious
    ${attachment}=    Create Dictionary    name=invoice.exe    content_type=application/octet-stream
    ${payload}=    Evaluate    {'message_id': 'robot-attach', 'sender': 'sender@example.test', 'subject': 'Attachment risk', 'body_preview': 'Review attached invoice', 'urls': [], 'attachments': [${attachment}]}
    ${verdict}=    Post Json    ${BASE_URL}/v1/email/scan    ${payload}
    ${status}=    Get From Dictionary    ${verdict}    status
    ${score}=    Get From Dictionary    ${verdict}    score
    Should Be Equal    ${status}    malicious
    Should Be Equal As Integers    ${score}    55
