import React from 'react';

import renderKeyValues from './GenericNotifier/GenericNotifier';

const formDescriptors = {
    authProviders: {
        oidc: [
            {
                label: 'Integration Name',
                jsonpath: 'name',
                type: 'text',
                placeholder: 'Auth0'
            },
            {
                label: 'Issuer',
                jsonpath: 'config.issuer',
                type: 'text',
                placeholder: 'your-tenant.auth0.com',
                immutable: true
            },
            {
                label: 'Client ID',
                jsonpath: 'config.client_id',
                type: 'text',
                placeholder: '',
                immutable: true
            },
            {
                label: 'Callback Mode',
                jsonpath: 'config.mode',
                type: 'select',
                options: [
                    { value: 'fragment', label: 'Fragment' },
                    { value: 'post', label: 'HTTP POST' }
                ],
                default: 'fragment',
                immutable: true
            }
        ],
        auth0: [
            {
                label: 'Integration Name',
                jsonpath: 'name',
                type: 'text',
                placeholder: 'Auth0'
            },
            {
                label: 'Auth0 Tenant',
                jsonpath: 'config.issuer',
                type: 'text',
                placeholder: 'your-tenant.auth0.com',
                immutable: true
            },
            {
                label: 'Client ID',
                jsonpath: 'config.client_id',
                type: 'text',
                placeholder: '',
                immutable: true
            },
            {
                jsonpath: 'config.mode',
                type: 'hidden',
                value: 'fragment'
            }
        ],
        saml: [
            {
                label: 'Integration Name',
                jsonpath: 'name',
                type: 'text',
                placeholder: 'Shibboleth'
            },
            {
                label: 'ServiceProvider Issuer',
                jsonpath: 'config.sp_issuer',
                type: 'text',
                placeholder: 'https://prevent.stackrox.io/',
                immutable: true
            },
            {
                html: (
                    <div className="flex mt-4 justify-center">
                        <div className="w-4/5 relative p-1 text-center text-separator">
                            Option 1: Dynamic Configuration
                        </div>
                    </div>
                ),
                type: 'html'
            },
            {
                label: 'IdP Metadata URL',
                jsonpath: 'config.idp_metadata_url',
                type: 'text',
                placeholder: 'https://idp.example.com/metadata',
                immutable: true
            },
            {
                html: (
                    <div className="flex mt-4 justify-center">
                        <div className="w-4/5 relative p-1 text-center text-separator">
                            Option 2: Static Configuration
                        </div>
                    </div>
                ),
                type: 'html'
            },
            {
                label: 'IdP Issuer',
                jsonpath: 'config.idp_issuer',
                type: 'text',
                placeholder: 'https://idp.example.com/',
                immutable: true
            },
            {
                label: 'IdP SSO URL',
                jsonpath: 'config.idp_sso_url',
                type: 'text',
                placeholder: 'https://idp.example.com/login',
                immutable: true
            },
            {
                label: 'IdP Certificate (PEM)',
                jsonpath: 'config.idp_cert_pem',
                type: 'textarea',
                placeholder:
                    '-----BEGIN CERTIFICATE-----\nYour certificate data\n-----END CERTIFICATE-----',
                immutable: true
            }
        ]
    },
    notifiers: {
        generic: [
            {
                label: 'Integration Name',
                jsonpath: 'name',
                type: 'text',
                placeholder: 'Generic Integration'
            },
            {
                label: 'Endpoint',
                jsonpath: 'generic.endpoint',
                type: 'text',
                placeholder: 'https://example.com/endpoint'
            },
            {
                label: 'Skip TLS Verification',
                jsonpath: 'generic.skipTLSVerify',
                type: 'toggle'
            },
            {
                label: 'Enable Audit Logging',
                jsonpath: 'generic.auditLoggingEnabled',
                type: 'toggle'
            },
            {
                label: 'CA Cert (optional)',
                jsonpath: 'generic.caCert',
                type: 'textarea',
                placeholder: ''
            },
            {
                label: 'Username (optional)',
                jsonpath: 'generic.username',
                type: 'text',
                placeholder: ''
            },
            {
                label: 'Password (optional)',
                jsonpath: 'generic.password',
                type: 'password',
                placeholder: ''
            },
            {
                label: 'Headers',
                jsonpath: 'generic.headers',
                type: 'list',
                listRender: renderKeyValues
            },
            {
                label: 'Extra Fields',
                jsonpath: 'generic.extraFields',
                type: 'list',
                listRender: renderKeyValues
            }
        ],
        jira: [
            {
                label: 'Integration Name',
                jsonpath: 'name',
                type: 'text',
                placeholder: 'Jira Integration'
            },
            {
                label: 'Username',
                jsonpath: 'jira.username',
                type: 'text',
                placeholder: 'user@example.com'
            },
            {
                label: 'Password',
                jsonpath: 'jira.password',
                type: 'password',
                placeholder: ''
            },
            {
                label: 'Issue Type',
                jsonpath: 'jira.issue_type',
                type: 'text',
                placeholder: 'Task, Sub-task, Story, Bug, or Epic'
            },
            {
                label: 'Jira URL',
                jsonpath: 'jira.url',
                type: 'text',
                placeholder: 'https://stack-rox.atlassian.net'
            },
            {
                label: 'Default Project',
                jsonpath: 'labelDefault',
                type: 'text',
                placeholder: 'PROJ'
            },
            {
                label: 'Label/Annotation Key for Project',
                jsonpath: 'labelKey',
                type: 'text'
            }
        ],
        email: [
            {
                label: 'Integration Name',
                jsonpath: 'name',
                type: 'text',
                placeholder: 'Email Integration'
            },
            {
                label: 'Email Server',
                jsonpath: 'email.server',
                type: 'text',
                placeholder: 'smtp.example.com:465'
            },
            {
                label: 'Username',
                jsonpath: 'email.username',
                type: 'text',
                placeholder: 'postmaster@example.com'
            },
            {
                label: 'Password',
                jsonpath: 'email.password',
                type: 'password'
            },
            {
                label: 'From',
                jsonpath: 'email.from',
                type: 'text',
                placeholder: 'StackRox'
            },
            {
                label: 'Sender',
                jsonpath: 'email.sender',
                type: 'text',
                placeholder: 'stackrox-notifier@example.com'
            },
            {
                label: 'Default Recipient',
                jsonpath: 'labelDefault',
                type: 'text',
                placeholder: 'stackrox-alerts@example.com'
            },
            {
                label: 'Label/Annotation Key for Recipient',
                jsonpath: 'labelKey',
                type: 'text',
                placeholder: 'email'
            },
            {
                label: 'Connect Without TLS (Unencrypted)',
                jsonpath: 'email.disableTLS',
                type: 'toggle',
                placeholder: ''
            },
            {
                label: 'Enable STARTTLS',
                jsonpath: 'email.useSTARTTLS',
                type: 'toggle',
                placeholder: ''
            }
        ],
        slack: [
            {
                label: 'Integration Name',
                jsonpath: 'name',
                type: 'text',
                placeholder: 'Slack Integration'
            },
            {
                label: 'Default Slack Webhook',
                jsonpath: 'labelDefault',
                type: 'text',
                placeholder: 'https://hooks.slack.com/services/EXAMPLE'
            },
            {
                label: 'Label/Annotation Key for Slack Webhook',
                jsonpath: 'labelKey',
                type: 'text',
                placeholder: 'slack'
            }
        ],
        cscc: [
            {
                label: 'Integration Name',
                jsonpath: 'name',
                type: 'text',
                placeholder: 'Cloud SCC'
            },
            {
                label: 'Cloud SCC Source ID',
                jsonpath: 'cscc.sourceId',
                type: 'text',
                placeholder: 'organizations/123/sources/456'
            },
            {
                label: 'Service Account Key (JSON)',
                jsonpath: 'cscc.serviceAccount',
                type: 'text',
                placeholder: '{"type": "service_account", "project_id": ...}'
            }
        ],
        splunk: [
            {
                label: 'Integration Name',
                jsonpath: 'name',
                type: 'text',
                placeholder: 'Splunk Integration'
            },
            {
                label: 'HTTP Event Collector URL',
                jsonpath: 'splunk.httpEndpoint',
                type: 'text',
                placeholder: 'https://localhost:8088'
            },
            {
                label: 'HTTP Event Collector Token',
                jsonpath: 'splunk.httpToken',
                type: 'password',
                placeholder: ''
            },
            {
                label: 'HEC Truncate Limit',
                jsonpath: 'splunk.truncate',
                type: 'number',
                placeholder: ''
            },
            {
                label: 'Disable TLS Certificate Validation (Insecure)',
                jsonpath: 'splunk.insecure',
                type: 'toggle',
                placeholder: ''
            },
            {
                label: 'Enable Audit Logging',
                jsonpath: 'splunk.auditLoggingEnabled',
                type: 'toggle'
            }
        ],
        sumologic: [
            {
                label: 'Integration Name',
                jsonpath: 'name',
                type: 'text',
                placeholder: 'SumoLogic Integration'
            },
            {
                label: 'HTTP Collector Source Address',
                jsonpath: 'sumologic.httpSourceAddress',
                type: 'text',
                placeholder: 'https://endpoint.sumologic.com/receiver/v1/http/<token>'
            },
            {
                label: 'Disable TLS Certificate Validation (Insecure)',
                jsonpath: 'sumologic.skipTLSVerify',
                type: 'toggle',
                placeholder: ''
            }
        ],
        pagerduty: [
            {
                label: 'Integration Name',
                jsonpath: 'name',
                type: 'text',
                placeholder: 'PagerDuty Integration'
            },
            {
                label: 'PagerDuty Integration Key',
                jsonpath: 'pagerduty.apiKey',
                type: 'password',
                placeholder: ''
            }
        ]
    },
    imageIntegrations: {
        tenable: [
            {
                label: 'Integration Name',
                jsonpath: 'name',
                type: 'text',
                placeholder: 'Tenable'
            },
            {
                label: 'Source Inputs',
                jsonpath: 'categories',
                type: 'multiselect',
                options: [
                    { value: 'REGISTRY', label: 'Registry' },
                    { value: 'SCANNER', label: 'Scanner' }
                ],
                placeholder: ''
            },
            {
                label: 'Access Key',
                jsonpath: 'tenable.accessKey',
                type: 'text',
                placeholder: ''
            },
            {
                label: 'Secret Key',
                jsonpath: 'tenable.secretKey',
                type: 'password',
                placeholder: ''
            }
        ],
        docker: [
            {
                label: 'Integration Name',
                jsonpath: 'name',
                type: 'text',
                placeholder: 'Docker Registry'
            },
            {
                label: 'Types',
                jsonpath: 'categories',
                type: 'multiselect',
                options: [{ value: 'REGISTRY', label: 'Registry', clearableValue: false }],
                placeholder: ''
            },
            {
                label: 'Endpoint',
                jsonpath: 'docker.endpoint',
                type: 'text',
                placeholder: 'registry-1.docker.io'
            },
            {
                label: 'Username',
                jsonpath: 'docker.username',
                type: 'text',
                placeholder: ''
            },
            {
                label: 'Password',
                jsonpath: 'docker.password',
                type: 'password',
                placeholder: ''
            },
            {
                label: 'Disable TLS Certificate Validation (Insecure)',
                jsonpath: 'docker.insecure',
                type: 'toggle',
                placeholder: ''
            }
        ],
        dtr: [
            {
                label: 'Integration Name',
                jsonpath: 'name',
                type: 'text',
                placeholder: 'Prod Docker Trusted Registry'
            },
            {
                label: 'Types',
                jsonpath: 'categories',
                type: 'multiselect',
                options: [
                    { value: 'REGISTRY', label: 'Registry' },
                    { value: 'SCANNER', label: 'Scanner' }
                ],
                placeholder: ''
            },
            {
                label: 'Endpoint',
                jsonpath: 'dtr.endpoint',
                type: 'text',
                placeholder: 'dtr.example.com'
            },
            {
                label: 'Username',
                jsonpath: 'dtr.username',
                type: 'text',
                placeholder: ''
            },
            {
                label: 'Password',
                jsonpath: 'dtr.password',
                type: 'password',
                placeholder: ''
            },
            {
                label: 'Disable TLS Certificate Validation (Insecure)',
                jsonpath: 'dtr.insecure',
                type: 'toggle',
                placeholder: ''
            }
        ],
        artifactory: [
            {
                label: 'Integration Name',
                jsonpath: 'name',
                type: 'text',
                placeholder: 'Artifactory'
            },
            {
                label: 'Types',
                jsonpath: 'categories',
                type: 'multiselect',
                options: [{ value: 'REGISTRY', label: 'Registry', clearableValue: false }],
                placeholder: ''
            },
            {
                label: 'Endpoint',
                jsonpath: 'docker.endpoint',
                type: 'text',
                placeholder: 'artifactory.example.com'
            },
            {
                label: 'Username',
                jsonpath: 'docker.username',
                type: 'text',
                placeholder: ''
            },
            {
                label: 'Password',
                jsonpath: 'docker.password',
                type: 'password',
                placeholder: ''
            }
        ],
        quay: [
            {
                label: 'Integration Name',
                jsonpath: 'name',
                type: 'text',
                placeholder: 'Quay'
            },
            {
                label: 'Types',
                jsonpath: 'categories',
                type: 'multiselect',
                options: [
                    { value: 'REGISTRY', label: 'Registry' },
                    { value: 'SCANNER', label: 'Scanner' }
                ],
                placeholder: ''
            },
            {
                label: 'Endpoint',
                jsonpath: 'quay.endpoint',
                type: 'text',
                placeholder: 'quay.io'
            },
            {
                label: 'OAuth Token',
                jsonpath: 'quay.oauthToken',
                type: 'text',
                placeholder: ''
            }
        ],
        clair: [
            {
                label: 'Integration Name',
                jsonpath: 'name',
                type: 'text',
                placeholder: 'Clair'
            },
            {
                label: 'Types',
                jsonpath: 'categories',
                type: 'multiselect',
                options: [{ value: 'SCANNER', label: 'Scanner', clearableValue: false }],
                placeholder: ''
            },
            {
                label: 'Endpoint',
                jsonpath: 'clair.endpoint',
                type: 'text',
                placeholder: 'https://clair.example.com'
            }
        ],
        clairify: [
            {
                label: 'Integration Name',
                jsonpath: 'name',
                type: 'text',
                placeholder: 'StackRox Scanner'
            },
            {
                label: 'Types',
                jsonpath: 'categories',
                type: 'multiselect',
                options: [{ value: 'SCANNER', label: 'Scanner', clearableValue: false }],
                placeholder: ''
            },
            {
                label: 'Endpoint',
                jsonpath: 'clairify.endpoint',
                type: 'text',
                placeholder: 'http://scanner.stackrox:8080'
            }
        ],
        google: [
            {
                label: 'Integration Name',
                jsonpath: 'name',
                type: 'text',
                placeholder: 'Google Registry and Scanner'
            },
            {
                label: 'Types',
                jsonpath: 'categories',
                type: 'multiselect',
                options: [
                    { value: 'REGISTRY', label: 'Registry' },
                    { value: 'SCANNER', label: 'Scanner' }
                ],
                placeholder: ''
            },
            {
                label: 'Registry Endpoint',
                jsonpath: 'google.endpoint',
                type: 'text',
                placeholder: 'gcr.io'
            },
            {
                label: 'Project',
                jsonpath: 'google.project',
                type: 'text',
                placeholder: ''
            },
            {
                label: 'Service Account Key (JSON)',
                jsonpath: 'google.serviceAccount',
                type: 'text',
                placeholder: ''
            }
        ],
        ecr: [
            {
                label: 'Integration Name',
                jsonpath: 'name',
                type: 'text',
                placeholder: 'AWS ECR'
            },
            {
                label: 'Types',
                jsonpath: 'categories',
                type: 'multiselect',
                options: [{ value: 'REGISTRY', label: 'Registry' }],
                placeholder: ''
            },
            {
                label: 'Registry ID',
                jsonpath: 'ecr.registryId',
                type: 'text',
                placeholder: '0123456789'
            },
            {
                label: 'Region',
                jsonpath: 'ecr.region',
                type: 'text',
                placeholder: 'us-west-2'
            },
            {
                label: 'Use Container IAM Role',
                jsonpath: 'ecr.useIam',
                type: 'toggle',
                placeholder: ''
            },
            {
                label: 'Access Key ID (required if not using IAM)',
                jsonpath: 'ecr.accessKeyId',
                type: 'password',
                placeholder: ''
            },
            {
                label: 'Secret Access Key (required if not using IAM)',
                jsonpath: 'ecr.secretAccessKey',
                type: 'password',
                placeholder: ''
            }
        ],
        nexus: [
            {
                label: 'Integration Name',
                jsonpath: 'name',
                type: 'text',
                placeholder: 'Nexus Registry'
            },
            {
                label: 'Types',
                jsonpath: 'categories',
                type: 'multiselect',
                options: [{ value: 'REGISTRY', label: 'Registry', clearableValue: false }],
                placeholder: ''
            },
            {
                label: 'Endpoint',
                jsonpath: 'docker.endpoint',
                type: 'text',
                placeholder: 'sonatype.nexus.com'
            },
            {
                label: 'Username',
                jsonpath: 'docker.username',
                type: 'text',
                placeholder: ''
            },
            {
                label: 'Password',
                jsonpath: 'docker.password',
                type: 'password',
                placeholder: ''
            },
            {
                label: 'Disable TLS Certificate Validation (Insecure)',
                jsonpath: 'docker.insecure',
                type: 'toggle',
                placeholder: ''
            }
        ],
        azure: [
            {
                label: 'Integration Name',
                jsonpath: 'name',
                type: 'text',
                placeholder: 'Azure Registry'
            },
            {
                label: 'Types',
                jsonpath: 'categories',
                type: 'multiselect',
                options: [{ value: 'REGISTRY', label: 'Registry', clearableValue: false }],
                placeholder: ''
            },
            {
                label: 'Endpoint',
                jsonpath: 'docker.endpoint',
                type: 'text',
                placeholder: '<registry>.azurecr.io'
            },
            {
                label: 'Username or App ID',
                jsonpath: 'docker.username',
                type: 'text',
                placeholder: ''
            },
            {
                label: 'Password or Service Principal Password',
                jsonpath: 'docker.password',
                type: 'password',
                placeholder: ''
            }
        ],
        anchore: [
            {
                label: 'Integration Name',
                jsonpath: 'name',
                type: 'text',
                placeholder: 'Anchore Scanner'
            },
            {
                label: 'Types',
                jsonpath: 'categories',
                type: 'multiselect',
                options: [{ value: 'SCANNER', label: 'Scanner', clearableValue: false }],
                placeholder: ''
            },
            {
                label: 'Endpoint',
                jsonpath: 'anchore.endpoint',
                type: 'text',
                placeholder: 'scanner.anchore.com'
            },
            {
                label: 'Username',
                jsonpath: 'anchore.username',
                type: 'text',
                placeholder: ''
            },
            {
                label: 'Password',
                jsonpath: 'anchore.password',
                type: 'password',
                placeholder: ''
            }
        ]
    },
    backups: {
        s3: [
            {
                label: 'Integration Name',
                jsonpath: 'name',
                type: 'text',
                placeholder: 'AWS S3'
            },
            {
                label: 'Backups To Retain',
                jsonpath: 'backupsToKeep',
                type: 'number',
                placeholder: '5'
            },
            {
                label: 'Schedule',
                jsonpath: 'schedule',
                type: 'schedule'
            },
            {
                label: 'Bucket',
                jsonpath: 's3.bucket',
                type: 'text',
                placeholder: 'stackrox-backups'
            },
            {
                label: 'Object Prefix (Optional)',
                jsonpath: 's3.objectPrefix',
                type: 'text',
                placeholder: ''
            },
            {
                label: 'Region',
                jsonpath: 's3.region',
                type: 'text',
                placeholder: 'us-west-2'
            },
            {
                label: 'Use Container IAM Role',
                jsonpath: 's3.useIam',
                type: 'toggle',
                placeholder: ''
            },
            {
                label: 'Access Key ID (required if not using IAM)',
                jsonpath: 's3.accessKeyId',
                type: 'password',
                placeholder: ''
            },
            {
                label: 'Secret Access Key (required if not using IAM)',
                jsonpath: 's3.secretAccessKey',
                type: 'password',
                placeholder: ''
            }
        ]
    }
};

export default formDescriptors;
