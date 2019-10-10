/* eslint-disable no-use-before-define */
import React from 'react';

import DateTimeField from './DateTimeField';

export default {
    title: 'DateTimeField',
    component: DateTimeField
};

export const basicDateTimeField = () => {
    const testDate = '2019-10-09T11:14:37.782231496Z';

    return <DateTimeField date={testDate} />;
};
