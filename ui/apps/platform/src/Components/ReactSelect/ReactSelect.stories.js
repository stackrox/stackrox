/* eslint-disable react-hooks/rules-of-hooks */
import React, { useState } from 'react';

import ReactSelect, { Creatable } from './ReactSelect';

export default {
    title: 'ReactSelect',
    component: ReactSelect,
};

function Parent({ children }) {
    // The height is for the drop-down menu (because overflow-hidden in Storybook).
    // The width is for the placeholder.
    return <div className="h-48 w-128">{children}</div>;
}

const lifecycleOptions = [
    {
        label: 'Build',
        value: 'BUILD',
    },
    {
        label: 'Deploy',
        value: 'DEPLOY',
    },
    {
        label: 'Runtime',
        value: 'RUNTIME',
    },
];

export const basicUsagePlaceholder = () => {
    const [lifecycles, setLifecycles] = useState('');

    return (
        <Parent>
            <ReactSelect
                id="lifecycleStages"
                name="lifecycleStages"
                options={lifecycleOptions}
                placeholder="Select Lifecycle Stage"
                onChange={setLifecycles}
                value={lifecycles}
            />
        </Parent>
    );
};

export const basicUsageValue = () => {
    const [lifecycles, setLifecycles] = useState('DEPLOY');

    return (
        <Parent>
            <ReactSelect
                id="lifecycleStages"
                name="lifecycleStages"
                options={lifecycleOptions}
                placeholder="Select Lifecycle Stage"
                onChange={setLifecycles}
                value={lifecycles}
            />
        </Parent>
    );
};

const existingPolicies = [
    {
        value: '93f4b2dd-ef5a-419e-8371-38aed480fb36',
        label: 'Fixable CVSS \u003e= 6 and Privileged',
    },
    {
        value: 'f09f8da1-6111-4ca0-8f49-294a76c65115',
        label: 'Fixable CVSS \u003e= 7',
    },
    {
        value: '13b4eddc-2619-4953-b1ee-4c762144ca1e',
        label: 'Images with no scans',
    },
];

export const asCreatable = () => {
    const [selectedPolicy, setSelectedPolicy] = useState(null);
    return (
        <Parent>
            <p className="bg-primary-300 mb-4 p-2 rounded-sm">
                Selected policy:{' '}
                {existingPolicies.find((options) => options.value === selectedPolicy)?.label ?? ''}
            </p>
            <Creatable
                key="policy"
                onChange={setSelectedPolicy}
                options={existingPolicies}
                placeholder="Type a name, or select an existing policy"
                value={selectedPolicy}
            />
        </Parent>
    );
};
