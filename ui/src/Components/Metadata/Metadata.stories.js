/* eslint-disable no-use-before-define */
import React from 'react';
import { MemoryRouter } from 'react-router-dom';

import Metadata from './Metadata';

export default {
    title: 'Metadata',
    component: Metadata
};

export const basicMetadata = () => {
    const title = 'CVSS Score Breakdown';
    const cvssScoreBreakdown = [
        {
            key: 'CVSS Score',
            value: 4.6
        },
        {
            key: 'Vector',
            value: 'AV:L/AC:L/Au:N/C:P/I:P'
        },
        {
            key: 'Impact Score',
            value: 2.3
        },
        {
            key: 'Exploitability Score',
            value: 7.5
        }
    ];

    return (
        <MemoryRouter>
            <Metadata title={title} keyValuePairs={cvssScoreBreakdown} />
        </MemoryRouter>
    );
};

export const withPopoverListsMetadata = () => {
    const keyValuePairs = [
        {
            key: 'Created',
            value: '10/12/2019 | 8:59:17AM'
        },
        {
            key: 'Deployment Type',
            value: 'Deployment'
        },
        {
            key: 'Replicas',
            value: 1
        }
    ];

    const labels = [
        {
            key: 'app',
            value: 'central',
            __typename: 'Label'
        },
        {
            key: 'app.kubernetes.io/name',
            value: 'stackrox',
            __typename: 'Label'
        }
    ];

    const annotations = [
        {
            key: 'email',
            value: 'support@stackrox.com',
            __typename: 'Label'
        },
        {
            key: 'owner',
            value: 'stackrox',
            __typename: 'Label'
        }
    ];

    return (
        <MemoryRouter>
            <Metadata keyValuePairs={keyValuePairs} labels={labels} annotations={annotations} />
        </MemoryRouter>
    );
};
