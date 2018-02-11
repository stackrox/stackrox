import React from 'react';

const tableColumnDescriptor = Object.freeze({
    authProviders: {
        auth0: [{ key: 'name', label: 'Name' }, { key: 'config.domain', label: 'Auth0 Domain' }]
    },
    notifiers: {
        slack: [
            { key: 'name', label: 'Name' },
            { key: 'config.webhook', label: 'Slack Webhook' },
            { key: 'config.channel', label: 'Slack Channel' }
        ],
        jira: [
            { key: 'name', label: 'Name' },
            { key: 'config.project', label: 'Project' },
            { key: 'config.issue_type', label: 'Issue Type' },
            {
                key: 'config.url',
                keyValueFunc: url => (
                    <a href={url} target="_blank">
                        {url}
                    </a>
                ),
                label: 'URL'
            }
        ],
        email: [
            { key: 'name', label: 'Name' },
            { key: 'config.recipient', label: 'Recipient' },
            { key: 'config.server', label: 'Server' }
        ]
    },
    scanners: {
        tenable: [
            { key: 'name', label: 'Name' },
            { key: 'endpoint', label: 'Scanner Endpoint' },
            { key: 'remote', label: 'Remote Endpoint' }
        ],
        dtr: [
            { key: 'name', label: 'Name' },
            { key: 'endpoint', label: 'Scanner Endpoint' },
            { key: 'remote', label: 'Remote Endpoint' }
        ]
    },
    registries: {
        docker: [
            { key: 'name', label: 'Name' },
            { key: 'endpoint', label: 'Scanner Endpoint' },
            { key: 'remote', label: 'Remote Endpoint' }
        ],
        tenable: [
            { key: 'name', label: 'Name' },
            { key: 'endpoint', label: 'Scanner Endpoint' },
            { key: 'remote', label: 'Remote Endpoint' }
        ],
        dtr: [
            { key: 'name', label: 'Name' },
            { key: 'endpoint', label: 'Scanner Endpoint' },
            { key: 'remote', label: 'Remote Endpoint' }
        ]
    }
});

export default tableColumnDescriptor;
