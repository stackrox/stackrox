/* eslint-disable no-nested-ternary */
import React from 'react';

import CollapsibleSection from 'Components/CollapsibleSection';
import CveType from 'Components/CveType';
import Metadata from 'Components/Metadata';
import ExternalLink from 'Components/PatternFly/IconText/ExternalLink';
import entityTypes from 'constants/entityTypes';
import { getDateTime } from 'utils/dateUtils';
import { isValidURL } from 'utils/urlUtils';
import RelatedEntitiesSideList from '../RelatedEntitiesSideList';

const emptyCve = {
    imageComponentCount: 0,
    nodeComponentCount: 0,
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
        imageComponentCount,
        nodeComponentCount,
    } = safeData;
    const operatingSystem = safeData?.operatingSystem;

    const linkToMoreInfo = isValidURL(link) ? (
        <ExternalLink>
            <a href={link} target="_blank" rel="noopener noreferrer">
                View Full CVE Description
            </a>
        </ExternalLink>
    ) : (
        <span>Full description unavailable</span>
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
            value: createdAt ? getDateTime(createdAt) : 'N/A',
        },
        {
            key: 'Published',
            value: publishedOn ? getDateTime(publishedOn) : 'N/A',
        },
        {
            key: 'Last modified',
            value: lastModified ? getDateTime(lastModified) : 'N/A',
        },
        {
            key: 'Scoring version',
            value: scoreVersion && `CVSS ${scoreVersion}`,
        },
    ];

    const splitCveType =
        imageComponentCount > 0
            ? entityTypes.IMAGE_CVE
            : nodeComponentCount > 0
              ? entityTypes.NODE_CVE
              : entityTypes.CLUSTER_CVE;
    const newEntityContext = { ...entityContext, [splitCveType]: cve };

    const cveType = Object.keys(newEntityContext).shift();
    let legacyTypeList = [];
    if (cveType === entityTypes.CLUSTER && splitCveType === entityTypes.CLUSTER_CVE) {
        legacyTypeList = vulnerabilityTypes;
    } else if (splitCveType === entityTypes.IMAGE_CVE || splitCveType === entityTypes.NODE_CVE) {
        legacyTypeList = [splitCveType];
    } else {
        legacyTypeList = [cveType];
    }

    const metaDataDetails = [
        {
            key: 'CVE',
            value: cve,
        },
        {
            key: 'Environment Impact',
            value: `${(envImpact * 100).toFixed(0)}%`,
        },
        {
            key: 'CVE Type',
            value: <CveType types={legacyTypeList} />,
        },
        {
            key: 'CVSS Score',
            value: cvss && cvss.toFixed(1),
        },
        {
            key: 'Fixability',
            value: isFixable ? 'Fixable' : 'Not fixable',
        },
    ];
    if (cveType === entityTypes.NODE_CVE || cveType === entityTypes.IMAGE_CVE) {
        metaDataDetails.push({
            key: 'Operating System',
            value: operatingSystem,
        });
    }

    return (
        <div className="flex h-full" data-testid="entity-overview">
            <div className="flex flex-col flex-grow min-w-0">
                <CollapsibleSection title="CVE Summary">
                    <div className="mx-4 grid-dense grid-auto-fit grid grid-gap-6 xxxl:grid-gap-8 lg:grid-columns-2 xl:grid-columns-3 mb-4">
                        <Metadata
                            className="h-full min-w-48 bg-base-100 pdf-page s-2"
                            headerComponents={linkToMoreInfo}
                            keyValuePairs={metaDataDetails}
                            description={summary || 'No description available.'}
                            title="Description & Details"
                        />
                        <Metadata
                            className="bg-base-100 min-h-48 pdf-page s-1"
                            keyValuePairs={cvssScoreBreakdown}
                            title="CVSS Score Breakdown"
                        />
                        <Metadata
                            className="bg-base-100 min-h-48 pdf-page s-1"
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
