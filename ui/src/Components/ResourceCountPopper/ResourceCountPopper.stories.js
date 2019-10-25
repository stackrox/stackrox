import React from 'react';

import ResourceCountPopper from './ResourceCountPopper';

export default {
    title: 'ResourceCountPopper',
    component: ResourceCountPopper
};

export const withData = () => {
    const data = [
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

    const label = 'Labels';

    return <ResourceCountPopper data={data} label={label} />;
};
