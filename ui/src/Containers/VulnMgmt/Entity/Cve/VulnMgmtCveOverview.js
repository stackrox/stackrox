import React, { useContext } from 'react';
import { ExternalLink } from 'react-feather';
import { format } from 'date-fns';
import pluralize from 'pluralize';

import CollapsibleSection from 'Components/CollapsibleSection';
import Metadata from 'Components/Metadata';
import LabelChip from 'Components/LabelChip';
import TileList from 'Components/TileList';
import Widget from 'Components/Widget';
import dateTimeFormat from 'constants/dateTimeFormat';
import entityTypes from 'constants/entityTypes';
import WorkflowStateMgr from 'modules/WorkflowStateManager';
import { generateURL } from 'modules/URLReadWrite';
import workflowStateContext from 'Containers/workflowStateContext';
import { getSeverityChipType } from 'utils/vulnerabilityUtils';

function getPushEntityType(workflowState, entityType) {
    const workflowStateMgr = new WorkflowStateMgr(workflowState);
    workflowStateMgr.pushList(entityType);
    const url = generateURL(workflowStateMgr.workflowState);

    return url;
}

const VulnMgmtCveOverview = ({ data }) => {
    const workflowState = useContext(workflowStateContext);

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
        scoreVersion,
        componentCount,
        imageCount,
        deploymentCount
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

    const tiles =
        deploymentCount || imageCount || componentCount
            ? [
                  {
                      count: deploymentCount,
                      label: pluralize('Deployment', deploymentCount),
                      url: getPushEntityType(workflowState, entityTypes.DEPLOYMENT)
                  },
                  {
                      count: imageCount,
                      label: pluralize('Image', imageCount),
                      url: getPushEntityType(workflowState, entityTypes.IMAGE)
                  },
                  {
                      count: componentCount,
                      label: pluralize('Component', componentCount),
                      url: getPushEntityType(workflowState, entityTypes.COMPONENT)
                  }
              ]
            : [];

    const severityStyle = getSeverityChipType(cvss);

    return (
        <div className="w-full h-full" id="capture-dashboard-stretch">
            <div className="flex h-full">
                <div className="flex flex-col flex-grow">
                    <CollapsibleSection title="CVE summary">
                        <div className="flex mb-4 pdf-page">
                            <Widget
                                header="Details"
                                headerComponents={linkToNVD}
                                className="mx-4 bg-base-100 h-48 mb-4 flex-grow"
                            >
                                <div className="flex flex-col w-full">
                                    <div className="bg-primary-200 text-2xl text-base-500 flex flex-col xl:flex-row items-start xl:items-center justify-between">
                                        <div className="w-full flex-grow p-4">
                                            <span>{cve}</span>
                                        </div>
                                        <div className="w-full flex border-t border-base-400 xl:border-t-0 justify-end items-center">
                                            <span className="px-6 py-4 border-base-400 border-l">
                                                <LabelChip
                                                    text={`CVSS ${cvss && cvss.toFixed(1)}`}
                                                    type={severityStyle}
                                                />
                                            </span>
                                            <span className="px-6 py-4 border-base-400 border-l">
                                                {isFixable ? (
                                                    <LabelChip text="Fixable" type="success" />
                                                ) : (
                                                    <LabelChip text="Not fixable" type="base" />
                                                )}
                                            </span>
                                        </div>
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

                <div className="bg-primary-300 h-full relative">
                    {/* TODO: decide if this should be added as custom tailwind class, or a "component" CSS class in app.css */}
                    <h2
                        style={{
                            position: 'relative',
                            left: '-0.5rem',
                            width: 'calc(100% + 0.5rem)'
                        }}
                        className="my-4 p-2 bg-primary-700 text-base text-base-100 rounded-l"
                    >
                        Related entities
                    </h2>
                    <TileList items={tiles} title="Contains" />
                </div>
            </div>
        </div>
    );
};

export default VulnMgmtCveOverview;
