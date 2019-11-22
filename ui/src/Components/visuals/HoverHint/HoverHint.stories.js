import React from 'react';

import HoverHint from './HoverHint';
import HoverHintListItem from '../HoverHintListItem';

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
            <HoverHintListItem key="categories" label="Category" value="Vulnerability Management" />
            <HoverHintListItem
                key="description"
                label="Description"
                value="Alert on deployments with a vulnerability with a CVSS &gt;= 7"
            />
            <HoverHintListItem
                key="latestViolation"
                label="Last violated"
                value="11/19/2019 11:51:59AM"
            />
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

export const withOptionalFooterAndSubtitle = () => {
    const tooltipBody = (
        <ul className="flex-1 list-reset border-base-300 overflow-hidden">
            <HoverHintListItem key="severity" label="Severity" value="Critical" />
            <HoverHintListItem key="riskScore" label="Risk Priority" value="8.9" />
            <HoverHintListItem key="weightedCvss" label="Weigthed CVSS" value="8.7" />
            <HoverHintListItem key="cves" label="CVEs" value="55 total, 20 fixable" />
        </ul>
    );

    const hintData = {
        title: 'jon-snow',
        subtitle: 'remote/stackrox',
        body: tooltipBody,
        top: 20,
        left: 10,
        footer: 'Scored using CVSS 3.0'
    };

    return (
        <div className="bg-base-300 relative min-h-55">
            <HoverHint
                title={hintData.title}
                subtitle={hintData.subtitle}
                body={hintData.body}
                top={hintData.top}
                left={hintData.left}
                footer={hintData.footer}
            />
        </div>
    );
};
