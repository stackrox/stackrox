import React from 'react';

import Labeled from './Labeled';

export default {
    title: 'Labeled',
    component: Labeled,
};

export const withTextLabelAndValue = () => <Labeled label="Label">Value</Labeled>;

export const withTextLabelAndEmptyValue = () => (
    <Labeled label="Does not render if value is empty string" />
);

export const withElementLabelAndValue = () => {
    const label = (
        <p>
            Details <i>(for geeks)</i>
        </p>
    );
    return (
        <Labeled label={label}>
            <pre>{JSON.stringify({ key1: 'value1', key2: 'value2', key3: 'value3' }, null, 2)}</pre>
        </Labeled>
    );
};
