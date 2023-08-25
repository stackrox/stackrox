import React from 'react';
import { ExternalLink } from 'react-feather';
import { format } from 'date-fns';

import CollapsibleSection from 'Components/CollapsibleSection';
import CveType from 'Components/CveType';
import Metadata from 'Components/Metadata';
import dateTimeFormat from 'constants/dateTimeFormat';
import entityTypes from 'constants/entityTypes';
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
    const operatingSystem = safeData?.operatingSystem;

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
        <span className="text-center text-base-600 bg-base-100 text-sm p-1">
            Full description unavailable
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

    const newEntityContext = { ...entityContext, [entityTypes.CVE]: cve };

    // TODO: change the CveType to handle one of the new split types: IMAGE_CVE, NODE_CVE, or CLUSTER_CVE
    //       but for now, we are going to translate the new data to the old type format
    const cveType = Object.keys(newEntityContext).shift();
    const legacyTypeList =
        cveType === entityTypes.CVE || cveType === entityTypes.CLUSTER_CVE
            ? vulnerabilityTypes
            : [cveType];

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
