import React from 'react';

import DetailedTooltipOverlay from './DetailedTooltipOverlay';
import HoverHintListItem from '../visuals/HoverHintListItem';

export default {
    title: 'DetailedTooltipOverlay',
    component: DetailedTooltipOverlay,
};

export const withTitleAndBody = () => {
    const tooltipData = {
        title: 'scanner',
        body: 'Weighted CVSS: 6.7',
    };

    return <DetailedTooltipOverlay title={tooltipData.title} body={tooltipData.body} />;
};

export const withOptionalFooter = () => {
    const tooltipBody = (
        <ul className="flex-1 border-base-300 overflow-hidden">
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

    const tooltipData = {
        title: 'scanner',
        body: tooltipBody,
        footer: 'Scored using CVSS 3.0',
    };

    return (
        <DetailedTooltipOverlay
            title={tooltipData.title}
            body={tooltipData.body}
            footer={tooltipData.footer}
        />
    );
};

export const withOptionalFooterAndSubtitle = () => {
    const tooltipBody = (
        <ul className="flex-1  border-base-300 overflow-hidden">
            <HoverHintListItem key="severity" label="Severity" value="Critical" />
            <HoverHintListItem key="riskScore" label="Risk Priority" value="8.9" />
            <HoverHintListItem key="weightedCvss" label="Weigthed CVSS" value="8.7" />
            <HoverHintListItem key="cves" label="CVEs" value="55 total, 20 fixable" />
        </ul>
    );

    const tooltipData = {
        title: 'jon-snow',
        subtitle: 'remote/stackrox',
        body: tooltipBody,
        footer: 'Scored using CVSS 3.0',
    };

    return (
        <DetailedTooltipOverlay
            title={tooltipData.title}
            subtitle={tooltipData.subtitle}
            body={tooltipData.body}
            footer={tooltipData.footer}
        />
    );
};
