import React from 'react';

import SectionHeaderInput from './SectionHeaderInput';

export default {
    title: 'SectionHeaderInput',
    component: SectionHeaderInput
};

export const withHeader = () => {
    return <SectionHeaderInput header="hello" />;
};
