import React from 'react';

const tableColumnDescriptor = Object.freeze({
    authProviders: {
        auth0: [
            { accessor: 'name', Header: 'Name' },
            { accessor: 'config.domain', Header: 'Auth0 Domain' }
        ]
    },
    notifiers: {
        slack: [
            { accessor: 'name', Header: 'Name' },
            { accessor: 'labelDefault', Header: 'Default Webhook', className: 'word-break' },
            { accessor: 'labelKey', Header: 'Webhook Label Key' }
        ],
        jira: [
            { accessor: 'name', Header: 'Name' },
            { accessor: 'labelDefault', Header: 'Default Project' },
            { accessor: 'labelKey', Header: 'Project Label Key' },
            {
                accessor: 'jira.url',
                keyValueFunc: url => (
                    <a href={url} target="_blank" rel="noopener noreferrer">
                        {url}
                    </a>
                ),
                Header: 'URL'
            }
        ],
        email: [
            { accessor: 'name', Header: 'Name' },
            { accessor: 'labelDefault', Header: 'Default Recipient' },
            { accessor: 'labelKey', Header: 'Recipient Label Key' },
            { accessor: 'email.server', Header: 'Server' }
        ],
        cscc: [
            { accessor: 'name', Header: 'Name' },
            { accessor: 'cscc.gcpOrgId', Header: 'Google Cloud Platform Org ID' }
        ]
    },
    imageIntegrations: {
        docker: [
            { accessor: 'name', Header: 'Name' },
            { accessor: 'docker.endpoint', Header: 'Endpoint' },
            { accessor: 'docker.username', Header: 'Username' }
        ],
        tenable: [{ accessor: 'name', Header: 'Name' }],
        dtr: [
            { accessor: 'name', Header: 'Name' },
            { accessor: 'dtr.endpoint', Header: 'Endpoint' },
            { accessor: 'dtr.username', Header: 'Username' }
        ],
        artifactory: [
            { accessor: 'name', Header: 'Name' },
            { accessor: 'docker.endpoint', Header: 'Endpoint' },
            { accessor: 'docker.username', Header: 'Username' }
        ],
        quay: [
            { accessor: 'name', Header: 'Name' },
            { accessor: 'quay.endpoint', Header: 'Endpoint' }
        ],
        clair: [
            { accessor: 'name', Header: 'Name' },
            { accessor: 'config.endpoint', Header: 'Endpoint' }
        ],
        clairify: [
            { accessor: 'name', Header: 'Name' },
            { accessor: 'clairify.endpoint', Header: 'Endpoint' }
        ],
        google: [
            { accessor: 'name', Header: 'Name' },
            { accessor: 'config.endpoint', Header: 'Endpoint' }
        ],
        ecr: [
            { accessor: 'name', Header: 'Name' },
            { accessor: 'ecr.registryId', Header: 'Registry ID' },
            { accessor: 'ecr.region', Header: 'Region' }
        ]
    }
});

export default tableColumnDescriptor;
