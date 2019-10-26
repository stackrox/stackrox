import React from 'react';

import RiskScore from 'Components/RiskScore';
import StatusChip from 'Components/StatusChip';
import TopCvssLabel from 'Components/TopCvssLabel';

import MetadataStatsList from './MetadataStatsList';

export default {
    title: 'MetadataStatsList',
    component: MetadataStatsList
};

export const withOneItem = () => {
    const oneItem = [<RiskScore score={3} />];

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
        <RiskScore score={5} />,
        <>
            <span className="pr-1">Policy status:</span>
            <StatusChip status="fail" />
        </>
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
        <RiskScore score={7} />,
        <>
            <span className="pr-1">Policy status:</span>
            <StatusChip status="fail" />
        </>,
        <TopCvssLabel cvss={8.2} expanded version="V3" />
    ];

    return (
        <div className="w-1/2 border">
            <div className="flex flex-col w-full">
                <MetadataStatsList statTiles={threeItems} />
            </div>
        </div>
    );
};
