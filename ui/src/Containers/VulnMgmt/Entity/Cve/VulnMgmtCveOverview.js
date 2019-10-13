import React from 'react';
import { ExternalLink } from 'react-feather';
import { format } from 'date-fns';

import CollapsibleSection from 'Components/CollapsibleSection';
import Metadata from 'Components/Metadata';
import LabelChip from 'Components/LabelChip';
import Widget from 'Components/Widget';
import dateTimeFormat from 'constants/dateTimeFormat';
import { getSeverityChipType } from 'utils/vulnerabilityUtils';

const VulnMgmtCveOverview = ({ data }) => {
    const {
        cve,
        cvss,
        vectors,
        isFixable,
        summary,
        link,
        lastScanned,
        published,
        lastModified,
        scoreVersion
    } = data;

    const linkToNVD = (
        <a
            href={link}
            className="btn-sm btn-base no-underline p-1"
            target="_blank"
            rel="noopener noreferrer nofollow"
        >
            <span className="pr-1">View on NVD Website</span>
            <ExternalLink size={16} />
        </a>
    );

    const cvssScoreBreakdown = [
        {
            key: 'CVSS Score',
            value: cvss && cvss.toFixed(1)
        },
        {
            key: 'Vector',
            value: vectors && vectors.vector
        },
        {
            key: 'Impact Score',
            value: vectors.impactScore && vectors.impactScore.toFixed(1)
        },
        {
            key: 'Exploitability Score',
            value: vectors.exploitabilityScore && vectors.exploitabilityScore.toFixed(1)
        }
    ];

    const scanningDetails = [
        {
            key: 'Scanned',
            value: lastScanned ? format(lastScanned, dateTimeFormat) : 'N/A'
        },
        {
            key: 'Published',
            value: published
        },
        {
            key: 'Last modified',
            value: lastModified
        },
        {
            key: 'Scoring version',
            value: scoreVersion && `CVSS ${scoreVersion}`
        }
    ];

    const severityStyle = getSeverityChipType(cvss);

    return (
        <div className="w-full" id="capture-dashboard-stretch">
            <CollapsibleSection title="CVE summary">
                <div className="flex mb-4 pdf-page">
                    <Widget
                        header="Details"
                        headerComponents={linkToNVD}
                        className="mx-4 bg-base-100 h-48 mb-4 flex-grow"
                    >
                        <div className="flex flex-col w-full">
                            <div className="bg-primary-200 text-2xl text-base-500 flex items-center justify-between">
                                <span className="flex-grow p-4">{cve}</span>
                                <span className="px-8 py-4 border-base-400 border-l">
                                    <LabelChip
                                        text={`CVSS ${cvss && cvss.toFixed(1)}`}
                                        type={severityStyle}
                                    />
                                </span>
                                <span className="px-8 py-4 border-base-400 border-l">
                                    {isFixable ? (
                                        <LabelChip text="Fixable" type="success" />
                                    ) : (
                                        <LabelChip text="Not fixable" type="base" />
                                    )}
                                </span>
                            </div>
                            <div className="p-4">{summary}</div>
                        </div>
                    </Widget>
                    <Metadata
                        className="mx-4 min-w-48 bg-base-100 h-48 mb-4"
                        keyValuePairs={cvssScoreBreakdown}
                        title="CVSS Score Breakdown"
                    />
                    <Metadata
                        className="mx-4 min-w-48 bg-base-100 h-48 mb-4"
                        keyValuePairs={scanningDetails}
                        title="Scanning Details"
                    />
                </div>
            </CollapsibleSection>
        </div>
    );
};

export default VulnMgmtCveOverview;
