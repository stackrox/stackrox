/* eslint-disable no-use-before-define */
import React from 'react';

import PageHeader from './PageHeader';

export default {
    title: 'PageHeader',
    component: PageHeader,
};

export const withHeaderText = () => {
    const header = 'nginx';

    return <PageHeader header={header} />;
};

export const withSubHeader = () => {
    const header = 'nginx';
    const subHeader = 'deployment';

    return <PageHeader header={header} subHeader={subHeader} />;
};

export const withSubHeaderNotCapitalized = () => {
    const header = 'nginx';
    const subHeader = 'deployment';

    return <PageHeader header={header} subHeader={subHeader} capitalize={false} />;
};
