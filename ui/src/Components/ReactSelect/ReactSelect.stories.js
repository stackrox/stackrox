/* eslint-disable react-hooks/rules-of-hooks */
import React, { useState } from 'react';

import ReactSelect, { Creatable } from './ReactSelect';

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

export default {
    title: 'ReactSelect',
    component: ReactSelect,
};

export const basicUsage = () => {
    const [lifecycles, setLifecycles] = useState([]);

    return (
        <ReactSelect
            id="lifecycleStages"
            name="lifecycleStages"
            options={lifecycleOptions}
            placeholder="Select Lifecycle Stage"
            onChange={setLifecycles}
            className="block w-full bg-base-100 border-base-300 text-base-600 z-1 focus:border-base-500"
            value={lifecycles}
        />
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
        <div>
            <Creatable
                key="policy"
                onChange={setSelectedPolicy}
                options={existingPolicies}
                placeholder="Type a name, or select an existing policy"
                value={selectedPolicy}
            />
            <p className="mt-4 p-2 bg-primary-300 rounded-sm">Selected policy: {selectedPolicy}</p>
        </div>
    );
};
