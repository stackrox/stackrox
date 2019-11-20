import React from 'react';

import HoverHint from './HoverHint';

export default {
    title: 'HoverHint',
    component: HoverHint
};

export const withTitleAndBody = () => {
    const hintData = {
        title: 'scanner',
        body: 'Weighted CVSS: 6.7'
    };

    return (
        <div className="bg-base-300 relative min-h-32">
            <HoverHint title={hintData.title} body={hintData.body} />
        </div>
    );
};

export const withOptionalOffsets = () => {
    const hintData = {
        title: 'scanner',
        body: 'Weighted CVSS: 6.7',
        top: 20,
        left: 10
    };

    return (
        <div className="bg-base-300 relative min-h-32">
            <HoverHint
                title={hintData.title}
                body={hintData.body}
                top={hintData.top}
                left={hintData.left}
            />
        </div>
    );
};

export const withOptionalFooter = () => {
    const tooltipBody = (
        <ul className="flex-1 list-reset border-base-300 overflow-hidden">
            <li className="py-1" key="categories">
                <span className="text-base-600 font-700 mr-2">Category:</span>
                <span className="font-600">Vulnerability Management</span>
            </li>
            <li className="py-1" key="description">
                <span className="text-base-600 font-700 mr-2">Description:</span>
                <span className="font-600">
                    Alert on deployments with a vulnerability with a CVSS &gt;= 7
                </span>
            </li>
            <li className="py-1" key="latestViolation">
                <span className="text-base-600 font-700 mr-2">Last violated:</span>
                <span className="font-600">11/19/2019 11:51:59AM</span>
            </li>
        </ul>
    );

    const hintData = {
        title: 'scanner',
        body: tooltipBody,
        top: 20,
        left: 10,
        footer: 'Scored using CVSS 3.0'
    };

    return (
        <div className="bg-base-300 relative min-h-55">
            <HoverHint
                title={hintData.title}
                body={hintData.body}
                top={hintData.top}
                left={hintData.left}
                footer={hintData.footer}
            />
        </div>
    );
};
