import React from 'react';
import { DndProvider } from 'react-dnd';
import Backend from 'react-dnd-html5-backend';

import PolicySection from './PolicySection';

export default {
    title: 'PolicySection',
    component: PolicySection
};

export const withHeader = () => {
    return (
        <DndProvider backend={Backend}>
            <PolicySection header="Policy Section 1" jsonpath="key1" />
        </DndProvider>
    );
};
