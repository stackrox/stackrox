import React from 'react';

import NameList from './NameList';

export default {
    title: 'Timeline Name List',
    component: NameList,
};

export const withoutChildren = () => {
    const names = [
        {
            type: 'graph-type-1',
            id: 'id-1',
            name: 'the podfather',
            subText: 'Started Jan 16, 2:45pm',
            hasChildren: false,
        },
        {
            type: 'graph-type-1',
            id: 'id-2',
            name: 'james pod',
            subText: 'Started Jan 20, 12:45am',
            hasChildren: false,
        },
    ];
    return <NameList names={names} />;
};

export const withChildren = () => {
    const names = [
        {
            type: 'graph-type-1',
            id: 'id-1',
            name: 'mary podpins',
            subText: 'Started Feb 12, 4:50pm',
            hasChildren: true,
        },
        {
            type: 'graph-type-1',
            id: 'id-2',
            name: 'the good, the bad, and the pod',
            subText: 'Started Mar, 1:50pm',
            hasChildren: true,
        },
    ];
    return <NameList names={names} />;
};
