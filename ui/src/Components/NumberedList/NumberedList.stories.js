import React from 'react';
import { MemoryRouter } from 'react-router-dom';

import SeverityStackedPill from 'Components/visuals/SeverityStackedPill';
import NumberedList from './NumberedList';

export default {
    title: 'NumberedList',
    component: NumberedList
};

export const withText = () => {
    const data = [
        { text: 'docker.io/library/nginx:1.7.9' },
        { text: 'k8s.gcr.io/prometheus-to-sd:v0.3.1' },
        { text: 'docker.io/stackrox/scanner:0.5.4' }
    ];
    return <NumberedList data={data} />;
};

export const withURL = () => {
    const data = [
        { text: 'docker.io/library/nginx:1.7.9', url: '/main/images/123' },
        { text: 'k8s.gcr.io/prometheus-to-sd:v0.3.1', url: '/main/images/456' },
        { text: 'docker.io/stackrox/scanner:0.5.4', url: '/main/images/789' }
    ];
    return (
        <MemoryRouter>
            <NumberedList data={data} />
        </MemoryRouter>
    );
};

export const withComponent = () => {
    const data = [
        {
            text: 'docker.io/library/nginx:1.7.9',
            component: <SeverityStackedPill low={10} medium={20} />
        },
        {
            text: 'k8s.gcr.io/prometheus-to-sd:v0.3.1',
            component: <SeverityStackedPill high={30} critical={5} />
        },
        {
            text: 'docker.io/stackrox/scanner:0.5.4',
            component: <SeverityStackedPill low={10} medium={20} high={30} critical={5} />
        }
    ];
    return (
        <MemoryRouter>
            <NumberedList data={data} />
        </MemoryRouter>
    );
};
