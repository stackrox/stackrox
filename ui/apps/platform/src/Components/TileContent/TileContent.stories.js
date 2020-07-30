import React from 'react';

import { Maximize2 } from 'react-feather';
import TileContent from './TileContent';

export default {
    title: 'TileContent',
    component: TileContent,
};

export const withSuperText = () => {
    const superText = '3';
    const text = 'Policy Violations';
    return <TileContent superText={superText} text={text} />;
};

export const withNumberSuperText = () => {
    const superText = 0;
    const text = 'Policy Violations';
    return <TileContent superText={superText} text={text} />;
};

export const withSubText = () => {
    const text = '6 Policies';
    const subText = '(2 Failing)';
    return <TileContent subText={subText} text={text} />;
};

export const withShort = () => {
    const text = '6 Policies';
    const subText = '(2 Failing)';
    return <TileContent subText={subText} text={text} short />;
};

export const withIcon = () => {
    const text = 'View Graph';
    return (
        <TileContent
            className="p-2"
            icon={<Maximize2 className="border border-primary-300 h-6 p-1 rounded-full w-6" />}
            text={text}
        />
    );
};
