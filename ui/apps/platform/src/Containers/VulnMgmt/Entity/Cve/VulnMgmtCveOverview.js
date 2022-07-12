import React from 'react';
import { ExternalLink } from 'react-feather';
import { format } from 'date-fns';

import CollapsibleSection from 'Components/CollapsibleSection';
import CveType from 'Components/CveType';
import Metadata from 'Components/Metadata';
import LabelChip from 'Components/LabelChip';
import Widget from 'Components/Widget';
import dateTimeFormat from 'constants/dateTimeFormat';
import entityTypes from 'constants/entityTypes';
import { getSeverityChipType } from 'utils/vulnerabilityUtils';
import { isValidURL } from 'utils/urlUtils';
import RelatedEntitiesSideList from '../RelatedEntitiesSideList';

const emptyCve = {
    componentCount: 0,
    cve: '',
    cvss: 0,
    deploymentCount: 0,
    envImpact: 0,
    fixedByVersion: '',
    imageCount: 0,
    isFixable: false,
    lastModified: '',
    createdAt: '',
    link: '',
    publishedOn: '',
    scoreVersion: '',
    summary: '',
    vectors: {},
    vulnerabilityTypes: [],
};

const VulnMgmtCveOverview = ({ data, entityContext }) => {
    // guard against incomplete GraphQL-cached data
    const safeData = { ...emptyCve, ...data };

    const {
        cve,
        cvss,
        envImpact,
        vectors,
        isFixable,
        summary,
        link,
        createdAt,
        publishedOn,
        lastModified,
        scoreVersion,
        vulnerabilityTypes,
    } = safeData;

    const linkToMoreInfo = isValidURL(link) ? (
        <a
            href={link}
            className="btn-sm btn-base no-underline p-1"
            target="_blank"
            rel="noopener noreferrer nofollow"
            data-testid="more-info-link"
        >
            <span className="pr-1">View Full CVE Description</span>
            <ExternalLink size={16} />
        </a>
    ) : (
        <span className="font-600 uppercase text-center text-base-600 bg-base-100 text-xs p-1">
            Full Description Unavailable
        </span>
    );

    const cvssScoreBreakdown = [
        {
            key: 'CVSS Score',
            value: cvss && cvss.toFixed(1),
        },
        {
            key: 'Vector',
            value: vectors && vectors.vector,
        },
        {
            key: 'Impact Score',
            value: vectors && vectors.impactScore && vectors.impactScore.toFixed(1),
        },
        {
            key: 'Exploitability Score',
            value: vectors && vectors.exploitabilityScore && vectors.exploitabilityScore.toFixed(1),
        },
    ];

    const scanningDetails = [
        {
            key: 'Discovered Time',
            value: createdAt ? format(createdAt, dateTimeFormat) : 'N/A',
        },
        {
            key: 'Published',
            value: publishedOn ? format(publishedOn, dateTimeFormat) : 'N/A',
        },
        {
            key: 'Last modified',
            value: lastModified ? format(lastModified, dateTimeFormat) : 'N/A',
        },
        {
            key: 'Scoring version',
            value: scoreVersion && `CVSS ${scoreVersion}`,
        },
    ];

    const severityStyle = getSeverityChipType(cvss);
    const newEntityContext = { ...entityContext, [entityTypes.CVE]: cve };

    // TODO: change the CveType to handle one of the new split types: IMAGE_CVE, NODE_CVE, or CLUSTER_CVE
    //       but for now, we are going to translate the new data to the old type format
    const cveType = Object.keys(newEntityContext).shift();
    const legacyTypeList =
        cveType === entityTypes.CVE || cveType === entityTypes.CLUSTER_CVE
            ? vulnerabilityTypes
            : [cveType];

    return (
        <div className="flex h-full" data-testid="entity-overview">
            <div className="flex flex-col flex-grow min-w-0">
                <CollapsibleSection title="CVE Summary">
                    <div className="mx-4 grid-dense grid-auto-fit grid grid-gap-6 xxxl:grid-gap-8 lg:grid-columns-2 xl:grid-columns-3 mb-4">
                        <Widget
                            header="Description & Details"
                            headerComponents={linkToMoreInfo}
                            className="bg-base-100 min-h-48 lg:s-2 pdf-page pdf-stretch"
                        >
                            <div className="flex flex-col w-full">
                                <div className="bg-tertiary-200 text-2xl text-base-500 flex flex-col md:flex-row items-start md:items-center justify-between">
                                    <div className="w-full flex-grow p-4">
                                        <span className="text-tertiary-800">{cve}</span>
                                    </div>
                                    <div className="w-full flex border-t border-base-400 md:border-t-0 justify-end items-center">
                                        {
                                            // eslint-disable-next-line eqeqeq
                                            envImpact == Number(envImpact) && (
                                                <span className="w-full md:w-auto p-4 border-base-400 text-base-600 border-l whitespace-nowrap">
                                                    <span>
                                                        {' '}
                                                        {`Env. Impact: ${(envImpact * 100).toFixed(
                                                            0
                                                        )}%`}
                                                    </span>
                                                </span>
                                            )
                                        }
                                        <span
                                            className="w-full md:w-auto p-4 border-base-400 border-l"
                                            data-testid="cve-type"
                                        >
                                            <CveType context="callout" types={legacyTypeList} />
                                        </span>
                                        <span className="w-full md:w-auto p-4 border-base-400 border-l">
                                            <LabelChip
                                                text={`CVSS ${cvss && cvss.toFixed(1)}`}
                                                type={severityStyle}
                                            />
                                        </span>
                                        <span className="w-full md:w-auto p-4 border-base-400 border-l">
                                            {isFixable ? (
                                                <LabelChip text="Fixable" type="success" />
                                            ) : (
                                                <LabelChip text="Not fixable" type="base" />
                                            )}
                                        </span>
                                    </div>
                                </div>
                                <div
                                    className="p-4 pb-12 leading-loose"
                                    data-testid="cve-description"
                                >
                                    {summary || 'No description available.'}
                                </div>
                            </div>
                        </Widget>
                        <Metadata
                            className="bg-base-100 min-h-48 pdf-page"
                            keyValuePairs={cvssScoreBreakdown}
                            title="CVSS Score Breakdown"
                        />
                        <Metadata
                            className="bg-base-100 min-h-48 pdf-page"
                            keyValuePairs={scanningDetails}
                            title="Scanning Details"
                        />
                    </div>
                </CollapsibleSection>
            </div>
            <RelatedEntitiesSideList
                entityType={cveType}
                entityContext={newEntityContext}
                data={safeData}
            />
        </div>
    );
};

export default VulnMgmtCveOverview;
