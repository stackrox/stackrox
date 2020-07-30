import React from 'react';

import RiskScore from 'Components/RiskScore';
import StatusChip from 'Components/StatusChip';
import TopCvssLabel from 'Components/TopCvssLabel';

import MetadataStatsList from './MetadataStatsList';

export default {
    title: 'MetadataStatsList',
    component: MetadataStatsList,
};

export const withOneItem = () => {
    const oneItem = [<RiskScore key="one" score={3} />];

    return (
        <div className="w-1/3 border">
            <div className="flex flex-col w-full">
                <MetadataStatsList statTiles={oneItem} />
            </div>
        </div>
    );
};

export const withTwoItems = () => {
    const twoItems = [
        <RiskScore key="one" score={5} />,
        <div key="two">
            <span className="pr-1">Policy status:</span>
            <StatusChip status="fail" />
        </div>,
    ];

    return (
        <div className="w-1/3 border">
            <div className="flex flex-col w-full">
                <MetadataStatsList statTiles={twoItems} />
            </div>
        </div>
    );
};

export const withThreeItems = () => {
    const threeItems = [
        <RiskScore key="one" score={7} />,
        <div key="two">
            <span className="pr-1">Policy status:</span>
            <StatusChip status="fail" />
        </div>,
        <TopCvssLabel key="three" cvss={8.2} expanded version="V3" />,
    ];

    return (
        <div className="w-1/2 border">
            <div className="flex flex-col w-full">
                <MetadataStatsList statTiles={threeItems} />
            </div>
        </div>
    );
};
