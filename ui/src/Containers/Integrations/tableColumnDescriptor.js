import React from 'react';
import dateFns from 'date-fns';
import dateTimeFormat from 'constants/dateTimeFormat';

const tableColumnDescriptor = Object.freeze({
    authProviders: {
        auth0: [{ key: 'name', label: 'Name' }, { key: 'config.domain', label: 'Auth0 Domain' }]
    },
    clusters: [
        { key: 'name', label: 'Name', className: 'word-break' },
        { key: 'preventImage', label: 'StackRox Image', className: 'word-break' },
        {
            key: 'lastContact',
            label: 'Last Check-In',
            keyValueFunc: date => {
                if (date) return dateFns.format(date, dateTimeFormat);
                return 'N/A';
            }
        }
    ],
    notifiers: {
        slack: [
            { key: 'name', label: 'Name' },
            { key: 'labelDefault', label: 'Default Webhook', className: 'word-break' },
            { key: 'labelKey', label: 'Webhook Label Key' }
        ],
        jira: [
            { key: 'name', label: 'Name' },
            { key: 'labelDefault', label: 'Default Project' },
            { key: 'labelKey', label: 'Project Label Key' },
            {
                key: 'jira.url',
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
            { key: 'labelDefault', label: 'Default Recipient' },
            { key: 'labelKey', label: 'Recipient Label Key' },
            { key: 'email.server', label: 'Server' }
        ],
        cscc: [
            { key: 'name', label: 'Name' },
            { key: 'cscc.gcpOrgId', label: 'Google Cloud Platform Org ID' }
        ]
    },
    imageIntegrations: {
        docker: [
            { key: 'name', label: 'Name' },
            { key: 'docker.endpoint', label: 'Endpoint' },
            { key: 'docker.username', label: 'Username' }
        ],
        tenable: [{ key: 'name', label: 'Name' }],
        dtr: [
            { key: 'name', label: 'Name' },
            { key: 'dtr.endpoint', label: 'Endpoint' },
            { key: 'dtr.username', label: 'Username' }
        ],
        artifactory: [
            { key: 'name', label: 'Name' },
            { key: 'docker.endpoint', label: 'Endpoint' },
            { key: 'docker.username', label: 'Username' }
        ],
        quay: [{ key: 'name', label: 'Name' }, { key: 'config.endpoint', label: 'Endpoint' }],
        clair: [{ key: 'name', label: 'Name' }, { key: 'config.endpoint', label: 'Endpoint' }],
        clairify: [{ key: 'name', label: 'Name' }, { key: 'clairify.endpoint', label: 'Endpoint' }],
        google: [{ key: 'name', label: 'Name' }, { key: 'config.endpoint', label: 'Endpoint' }]
    }
});

export default tableColumnDescriptor;
