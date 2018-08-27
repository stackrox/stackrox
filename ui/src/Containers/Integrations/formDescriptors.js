const formDescriptors = {
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
                key: 'jira.username',
                type: 'text',
                placeholder: 'user@example.com'
            },
            {
                label: 'Password',
                key: 'jira.password',
                type: 'password',
                placeholder: ''
            },
            {
                label: 'Issue Type',
                key: 'jira.issue_type',
                type: 'text',
                placeholder: 'Task, Sub-task, Story, Bug, or Epic'
            },
            {
                label: 'Jira URL',
                key: 'jira.url',
                type: 'text',
                placeholder: 'https://stack-rox.atlassian.net'
            },
            {
                label: 'Default Project',
                key: 'labelDefault',
                type: 'text',
                placeholder: 'PROJ'
            },
            {
                label: 'Label/Annotation Key for Project',
                key: 'labelKey',
                type: 'text'
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
                key: 'email.server',
                type: 'text',
                placeholder: 'smtp.example.com:465'
            },
            {
                label: 'Username',
                key: 'email.username',
                type: 'text',
                placeholder: 'postmaster@example.com'
            },
            {
                label: 'Password',
                key: 'email.password',
                type: 'password'
            },
            {
                label: 'Sender',
                key: 'email.sender',
                type: 'text',
                placeholder: 'prevent-notifier@example.com'
            },
            {
                label: 'Default Recipient',
                key: 'labelDefault',
                type: 'text',
                placeholder: 'prevent-alerts@example.com'
            },
            {
                label: 'Label/Annotation Key for Recipient',
                key: 'labelKey',
                type: 'text',
                placeholder: 'email'
            },
            {
                label: 'Disable TLS',
                key: 'email.tls',
                type: 'select',
                options: [{ label: 'On', value: 'true' }, { label: 'Off', value: 'false' }],
                placeholder: 'Disable TLS?'
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
                label: 'Default Slack Webhook',
                key: 'labelDefault',
                type: 'text',
                placeholder: 'https://hooks.slack.com/services/EXAMPLE'
            },
            {
                label: 'Label/Annotation Key for Slack Webhook',
                key: 'labelKey',
                type: 'text',
                placeholder: 'slack'
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
                key: 'cscc.gcpOrgId',
                type: 'text',
                placeholder: ''
            },
            {
                label: 'GCP Project',
                key: 'cscc.gcpProject',
                type: 'text',
                placeholder: ''
            },
            {
                label: 'Service Account Key (JSON)',
                key: 'cscc.serviceAccount',
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
                key: 'docker.endpoint',
                type: 'text',
                placeholder: 'registry-1.docker.io'
            },
            {
                label: 'Username',
                key: 'docker.username',
                type: 'text',
                placeholder: ''
            },
            {
                label: 'Password',
                key: 'docker.password',
                type: 'password',
                placeholder: ''
            },
            {
                label: 'Disable TLS Certificate Validation (Insecure)',
                key: 'docker.insecure',
                type: 'checkbox',
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
                label: 'Disable TLS Certificate Validation (Insecure)',
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
                key: 'docker.endpoint',
                type: 'text',
                placeholder: 'artifactory.example.com'
            },
            {
                label: 'Username',
                key: 'docker.username',
                type: 'text',
                placeholder: ''
            },
            {
                label: 'Password',
                key: 'docker.password',
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
                key: 'quay.endpoint',
                type: 'text',
                placeholder: 'quay.io'
            },
            {
                label: 'OAuth Token',
                key: 'quay.oauthToken',
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
        clairify: [
            {
                label: 'Integration Name',
                key: 'name',
                type: 'text',
                placeholder: 'Clairify'
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
                key: 'clairify.endpoint',
                type: 'text',
                placeholder: 'http://clairify.stackrox:8080'
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
        ],
        ecr: [
            {
                label: 'Integration Name',
                key: 'name',
                type: 'text',
                placeholder: 'AWS ECR'
            },
            {
                label: 'Types',
                key: 'categories',
                type: 'multiselect',
                options: [{ value: 'REGISTRY', label: 'Registry' }],
                placeholder: ''
            },
            {
                label: 'Registry ID',
                key: 'ecr.registryId',
                type: 'text',
                placeholder: '0123456789'
            },
            {
                label: 'Region',
                key: 'ecr.region',
                type: 'text',
                placeholder: 'us-west-2'
            },
            {
                label: 'Access Key ID',
                key: 'ecr.accessKeyId',
                type: 'password',
                placeholder: ''
            },
            {
                label: 'Secret Access Key',
                key: 'ecr.secretAccessKey',
                type: 'password',
                placeholder: ''
            }
        ]
    }
};

export default formDescriptors;
