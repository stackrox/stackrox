/* eslint-disable react-hooks/rules-of-hooks */
import React, { useState } from 'react';

import MultiSelect from './MultiSelect';

const lifecycleOptions = [
    {
        label: 'Build',
        value: 'BUILD'
    },
    {
        label: 'Deploy',
        value: 'DEPLOY'
    },
    {
        label: 'Runtime',
        value: 'RUNTIME'
    }
];

export default {
    title: 'MultiSelect',
    component: MultiSelect
};

export const basicUsage = () => {
    const [lifecycles, setLifecycles] = useState([]);

    return (
        <MultiSelect
            id="lifecycleStages"
            name="lifecycleStages"
            options={lifecycleOptions}
            placeholder="Select Lifecycle Stages"
            onChange={setLifecycles}
            className="block w-full bg-base-100 border-base-300 text-base-600 z-1 focus:border-base-500"
            value={lifecycles}
        />
    );
};
