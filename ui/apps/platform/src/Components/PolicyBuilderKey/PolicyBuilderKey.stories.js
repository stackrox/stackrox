import React from 'react';
import { DndProvider } from 'react-dnd';
import { HTML5Backend } from 'react-dnd-html5-backend';

import PolicyBuilderKey from './PolicyBuilderKey';

export default {
    title: 'PolicyBuilderKey',
    component: PolicyBuilderKey,
};

export const withLabel = () => {
    return (
        <DndProvider backend={HTML5Backend}>
            <PolicyBuilderKey label="Image tags" jsonpath="key1" />
        </DndProvider>
    );
};
