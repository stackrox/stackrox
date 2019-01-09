import React from 'react';

const tableColumnDescriptor = Object.freeze({
    authProviders: {
        oidc: [
            { accessor: 'name', Header: 'Name' },
            { accessor: 'config.issuer', Header: 'Issuer' }
        ],
        auth0: [
            { accessor: 'name', Header: 'Name' },
            { accessor: 'config.issuer', Header: 'Auth0 Tenant' }
        ],
        saml: [
            { accessor: 'name', Header: 'Name' },
            { accessor: 'config.idp_issuer', Header: 'Issuer' }
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
            { accessor: 'cscc.sourceId', Header: 'Google Cloud SCC Source ID' }
        ],
        splunk: [
            { accessor: 'name', Header: 'Name' },
            {
                accessor: 'splunk.httpEndpoint',
                keyValueFunc: url => (
                    <a href={url} target="_blank" rel="noopener noreferrer">
                        {url}
                    </a>
                ),
                Header: 'URL'
            }
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
            { accessor: 'clair.endpoint', Header: 'Endpoint' }
        ],
        clairify: [
            { accessor: 'name', Header: 'Name' },
            { accessor: 'clairify.endpoint', Header: 'Endpoint' }
        ],
        google: [
            { accessor: 'name', Header: 'Name' },
            { accessor: 'google.endpoint', Header: 'Endpoint' }
        ],
        ecr: [
            { accessor: 'name', Header: 'Name' },
            { accessor: 'ecr.registryId', Header: 'Registry ID' },
            { accessor: 'ecr.region', Header: 'Region' }
        ],
        nexus: [
            { accessor: 'name', Header: 'Name' },
            { accessor: 'docker.endpoint', Header: 'Endpoint' },
            { accessor: 'docker.username', Header: 'Username' }
        ]
    }
});

export default tableColumnDescriptor;
