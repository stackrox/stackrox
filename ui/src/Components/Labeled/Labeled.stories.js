import React from 'react';

import Labeled from './Labeled';

export default {
    title: 'Labeled',
    component: Labeled
};

export const withTextLabelAndValue = () => <Labeled label="Label">Value</Labeled>;

export const withTextLabelAndInput = () => (
    <Labeled label="Enter value">
        <input className="border-2" />
    </Labeled>
);

export const withElementLabelAndInput = () => {
    const label = (
        <p>
            Important thing <i className="text-base-500">(required)</i>
        </p>
    );
    return (
        <Labeled label={label}>
            <input className="border-2" />
        </Labeled>
    );
};
