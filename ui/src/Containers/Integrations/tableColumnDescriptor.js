import React from 'react';
import dateFns from 'date-fns';

const tableColumnDescriptor = Object.freeze({
    authProviders: {
        auth0: [{ key: 'name', label: 'Name' }, { key: 'config.domain', label: 'Auth0 Domain' }]
    },
    clusters: [
        { key: 'name', label: 'Name' },
        { key: 'preventImage', label: 'StackRox Image' },
        {
            key: 'lastContact',
            label: 'Last Check-In',
            keyValueFunc: date => {
                if (date) return dateFns.format(date, 'MM/DD/YYYY h:mm:ss A');
                return 'N/A';
            }
        }
    ],
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
        ],
        cscc: [
            { key: 'name', label: 'Name' },
            { key: 'config.gcpOrgID', label: 'Google Cloud Platform Org ID' }
        ]
    },
    imageIntegrations: {
        docker: [
            { key: 'name', label: 'Name' },
            { key: 'config.endpoint', label: 'Endpoint' },
            { key: 'config.username', label: 'Username' }
        ],
        tenable: [{ key: 'name', label: 'Name' }],
        dtr: [
            { key: 'name', label: 'Name' },
            { key: 'config.endpoint', label: 'Endpoint' },
            { key: 'config.username', label: 'Username' }
        ],
        artifactory: [
            { key: 'name', label: 'Name' },
            { key: 'config.endpoint', label: 'Endpoint' },
            { key: 'config.username', label: 'Username' }
        ],
        quay: [{ key: 'name', label: 'Name' }, { key: 'config.endpoint', label: 'Endpoint' }],
        clair: [{ key: 'name', label: 'Name' }, { key: 'config.endpoint', label: 'Endpoint' }],
        google: [{ key: 'name', label: 'Name' }, { key: 'config.endpoint', label: 'Endpoint' }]
    }
});

export default tableColumnDescriptor;
