import React from 'react';
import { DndProvider } from 'react-dnd';
import Backend from 'react-dnd-html5-backend';
import { FieldArray } from 'redux-form';

import PolicyBuilderKeys from 'Components/PolicyBuilderKeys';
import PolicySections from './PolicySections';
import { policyConfiguration } from './descriptors';

function BooleanPolicySection() {
    return (
        <DndProvider backend={Backend}>
            <div className="w-full flex">
                <FieldArray name="policy_sections" component={PolicySections} />
                <PolicyBuilderKeys keys={policyConfiguration.descriptor} />
            </div>
        </DndProvider>
    );
}

export default BooleanPolicySection;
