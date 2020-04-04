import React from 'react';
import { DndProvider } from 'react-dnd';
import Backend from 'react-dnd-html5-backend';

import { policyConfiguration } from 'Containers/Policies/Wizard/Form/descriptors';
import PolicyBuilderKeys from './PolicyBuilderKeys';

export default {
    title: 'PolicyBuilderKeys',
    component: PolicyBuilderKeys
};

export const withPolicyDescriptors = () => {
    return (
        <DndProvider backend={Backend}>
            <PolicyBuilderKeys keys={policyConfiguration.descriptor} />
        </DndProvider>
    );
};
