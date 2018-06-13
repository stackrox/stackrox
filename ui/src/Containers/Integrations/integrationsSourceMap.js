const sourceMap = {
    authProviders: {
        auth0: [
            {
                label: 'Integration Name',
                key: 'name',
                type: 'text',
                placeholder: 'Auth0'
            },
            {
                label: 'Domain',
                key: 'config.domain',
                type: 'text',
                placeholder: 'your-tenant.auth0.com'
            },
            {
                label: 'Client ID',
                key: 'config.client_id',
                type: 'text',
                placeholder: ''
            },
            {
                label: 'Audience',
                key: 'config.audience',
                type: 'text',
                placeholder: 'prevent.stackrox.io'
            }
        ]
    },
    notifiers: {
        jira: [
            {
                label: 'Integration Name',
                key: 'name',
                type: 'text',
                placeholder: 'Jira Integration'
            },
            {
                label: 'Username',
                key: 'config.username',
                type: 'text',
                placeholder: 'user@example.com'
            },
            {
                label: 'Password',
                key: 'config.password',
                type: 'password',
                placeholder: ''
            },
            {
                label: 'Project Key',
                key: 'config.project',
                type: 'text',
                placeholder: 'PROJ'
            },
            {
                label: 'Issue Type',
                key: 'config.issue_type',
                type: 'text',
                placeholder: 'Task, Sub-task, Story, Bug, or Epic'
            },
            {
                label: 'Jira URL',
                key: 'config.url',
                type: 'text',
                placeholder: 'https://example.atlassian.net'
            }
        ],
        email: [
            {
                label: 'Integration Name',
                key: 'name',
                type: 'text',
                placeholder: 'Email Integration'
            },
            {
                label: 'Email Server',
                key: 'config.server',
                type: 'text',
                placeholder: 'smtp.example.com:465'
            },
            {
                label: 'Username',
                key: 'config.username',
                type: 'text',
                placeholder: 'postmaster@example.com'
            },
            {
                label: 'Password',
                key: 'config.password',
                type: 'password'
            },
            {
                label: 'Sender',
                key: 'config.sender',
                type: 'text',
                placeholder: 'prevent-notifier@example.com'
            },
            {
                label: 'Recipient',
                key: 'config.recipient',
                type: 'text',
                placeholder: 'prevent-alerts@example.com'
            },
            {
                label: 'TLS',
                key: 'config.tls',
                type: 'select',
                options: [{ label: 'On', value: 'true' }, { label: 'Off', value: 'false' }],
                placeholder: 'Enable TLS?'
            }
        ],
        slack: [
            {
                label: 'Integration Name',
                key: 'name',
                type: 'text',
                placeholder: 'Slack Integration'
            },
            {
                label: 'Slack Webhook',
                key: 'config.webhook',
                type: 'text',
                placeholder: 'https://hooks.slack.com/services/EXAMPLE'
            },
            {
                label: 'Slack Channel',
                key: 'config.channel',
                type: 'text',
                placeholder: '#slack-channel'
            }
        ],
        cscc: [
            {
                label: 'Integration Name',
                key: 'name',
                type: 'text',
                placeholder: 'CSCC'
            },
            {
                label: 'GCP Organization ID Number',
                key: 'config.gcpOrgID',
                type: 'text',
                placeholder: ''
            },
            {
                label: 'GCP Project',
                key: 'config.gcpProject',
                type: 'text',
                placeholder: ''
            },
            {
                label: 'Service Account Key (JSON)',
                key: 'config.serviceAccount',
                type: 'text',
                placeholder: ''
            }
        ]
    },
    imageIntegrations: {
        tenable: [
            {
                label: 'Integration Name',
                key: 'name',
                type: 'text',
                placeholder: 'Tenable'
            },
            {
                label: 'Source Inputs',
                key: 'categories',
                type: 'multiselect',
                options: [
                    { value: 'REGISTRY', label: 'Registry' },
                    { value: 'SCANNER', label: 'Scanner' }
                ],
                placeholder: ''
            },
            {
                label: 'Access Key',
                key: 'config.accessKey',
                type: 'text',
                placeholder: ''
            },
            {
                label: 'Secret Key',
                key: 'config.secretKey',
                type: 'text',
                placeholder: ''
            }
        ],
        docker: [
            {
                label: 'Integration Name',
                key: 'name',
                type: 'text',
                placeholder: 'Docker Registry'
            },
            {
                label: 'Types',
                key: 'categories',
                type: 'multiselect',
                options: [{ value: 'REGISTRY', label: 'Registry', clearableValue: false }],
                placeholder: ''
            },
            {
                label: 'Endpoint',
                key: 'config.endpoint',
                type: 'text',
                placeholder: 'registry-1.docker.io'
            },
            {
                label: 'Username',
                key: 'config.username',
                type: 'text',
                placeholder: ''
            },
            {
                label: 'Password',
                key: 'config.password',
                type: 'password',
                placeholder: ''
            }
        ],
        dtr: [
            {
                label: 'Integration Name',
                key: 'name',
                type: 'text',
                placeholder: 'Prod Docker Trusted Registry'
            },
            {
                label: 'Types',
                key: 'categories',
                type: 'multiselect',
                options: [
                    { value: 'REGISTRY', label: 'Registry' },
                    { value: 'SCANNER', label: 'Scanner' }
                ],
                placeholder: ''
            },
            {
                label: 'Endpoint',
                key: 'dtr.endpoint',
                type: 'text',
                placeholder: 'dtr.example.com'
            },
            {
                label: 'Username',
                key: 'dtr.username',
                type: 'text',
                placeholder: ''
            },
            {
                label: 'Password',
                key: 'dtr.password',
                type: 'password',
                placeholder: ''
            },
            {
                label: 'Insecure',
                key: 'dtr.insecure',
                type: 'checkbox',
                placeholder: ''
            }
        ],
        artifactory: [
            {
                label: 'Integration Name',
                key: 'name',
                type: 'text',
                placeholder: 'Artifactory'
            },
            {
                label: 'Types',
                key: 'categories',
                type: 'multiselect',
                options: [{ value: 'REGISTRY', label: 'Registry', clearableValue: false }],
                placeholder: ''
            },
            {
                label: 'Endpoint',
                key: 'config.endpoint',
                type: 'text',
                placeholder: 'artifactory.example.com'
            },
            {
                label: 'Username',
                key: 'config.username',
                type: 'text',
                placeholder: ''
            },
            {
                label: 'Password',
                key: 'config.password',
                type: 'password',
                placeholder: ''
            }
        ],
        quay: [
            {
                label: 'Integration Name',
                key: 'name',
                type: 'text',
                placeholder: 'Quay'
            },
            {
                label: 'Types',
                key: 'categories',
                type: 'multiselect',
                options: [
                    { value: 'REGISTRY', label: 'Registry' },
                    { value: 'SCANNER', label: 'Scanner' }
                ],
                placeholder: ''
            },
            {
                label: 'Endpoint',
                key: 'config.endpoint',
                type: 'text',
                placeholder: 'quay.io'
            },
            {
                label: 'OAuth Token',
                key: 'config.oauthToken',
                type: 'text',
                placeholder: ''
            }
        ],
        clair: [
            {
                label: 'Integration Name',
                key: 'name',
                type: 'text',
                placeholder: 'Clair'
            },
            {
                label: 'Types',
                key: 'categories',
                type: 'multiselect',
                options: [{ value: 'SCANNER', label: 'Scanner', clearableValue: false }],
                placeholder: ''
            },
            {
                label: 'Endpoint',
                key: 'config.endpoint',
                type: 'text',
                placeholder: 'https://clair.example.com'
            }
        ],
        google: [
            {
                label: 'Integration Name',
                key: 'name',
                type: 'text',
                placeholder: 'Google Registry and Scanner'
            },
            {
                label: 'Types',
                key: 'categories',
                type: 'multiselect',
                options: [
                    { value: 'REGISTRY', label: 'Registry' },
                    { value: 'SCANNER', label: 'Scanner' }
                ],
                placeholder: ''
            },
            {
                label: 'Registry Endpoint',
                key: 'config.endpoint',
                type: 'text',
                placeholder: 'gcr.io'
            },
            {
                label: 'Project',
                key: 'config.project',
                type: 'text',
                placeholder: ''
            },
            {
                label: 'Service Account Key (JSON)',
                key: 'config.serviceAccount',
                type: 'text',
                placeholder: ''
            }
        ]
    }
};

export default sourceMap;
