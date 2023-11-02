import React, { ReactElement, useState } from 'react';
import { gql, useQuery } from '@apollo/client';
import Raven from 'raven-js';
import { Alert, Skeleton, Split, SplitItem } from '@patternfly/react-core';

import {
    clustersBasePath,
    configManagementPath,
    urlEntityListTypes,
    violationsBasePath,
    vulnManagementImagesPath,
} from 'routePaths';
import { resourceTypes } from 'constants/entityTypes';

import SummaryCount from './SummaryCount';

export type SummaryCountsResponse = {
    clusterCount: number;
    nodeCount: number;
    violationCount: number;
    deploymentCount: number;
    imageCount: number;
    secretCount: number;
};

const tileResources = ['Cluster', 'Node', 'Alert', 'Deployment', 'Image', 'Secret'] as const;
type TileResource = (typeof tileResources)[number];

const dataKey: Record<TileResource, string> = {
    Cluster: 'clusterCount',
    Node: 'nodeCount',
    Alert: 'violationCount',
    Deployment: 'deploymentCount',
    Image: 'imageCount',
    Secret: 'secretCount',
};

// According to current minimalist philosophy, ignore that routes might have additional resource requirements.
const tileLinks: Record<TileResource, string> = {
    Cluster: clustersBasePath,
    Node: `${configManagementPath}/${urlEntityListTypes[resourceTypes.NODE]}`,
    Alert: violationsBasePath,
    Deployment: `${configManagementPath}/${urlEntityListTypes[resourceTypes.DEPLOYMENT]}`,
    Image: vulnManagementImagesPath,
    Secret: `${configManagementPath}/${urlEntityListTypes[resourceTypes.SECRET]}`,
};

const tileNouns: Record<TileResource, string> = {
    Cluster: 'Cluster',
    Node: 'Node',
    Alert: 'Violation',
    Deployment: 'Deployment',
    Image: 'Image',
    Secret: 'Secret',
};

const locale = window.navigator.language ?? 'en-US';
const dateFormatter = new Intl.DateTimeFormat(locale);
const timeFormatter = new Intl.DateTimeFormat(locale, { hour: 'numeric', minute: 'numeric' });

export type SummaryCountsProps = {
    hasReadAccessForResource: Record<TileResource, boolean>;
};

function SummaryCounts({ hasReadAccessForResource }: SummaryCountsProps): ReactElement {
    const tileResourcesQuery = tileResources
        .filter((tileResource) => hasReadAccessForResource[tileResource])
        .map((tileResource) => dataKey[tileResource])
        .join('\n');
    const query = gql`
        query summary_counts {
            ${tileResourcesQuery}
        }
    `;

    const [lastUpdate, setLastUpdate] = useState<Date>(new Date());
    console.log('SummaryCoutns', query);
    const { loading, error, data } = useQuery<SummaryCountsResponse>(query, {
        errorPolicy: 'all',
        // fetchPolicy: 'network-only',
        // onCompleted: () => setLastUpdate(new Date()),
    });

    if (loading) {
        return (
            <Skeleton
                height="32px"
                className="pf-u-m-md"
                screenreaderText="Loading system summary counts"
            />
        );
    }

    console.log(error, query);

    if (error || !data) {
        Raven.captureException(error);
        return (
            <Alert
                isInline
                variant="warning"
                title="There was an error loading system summary counts"
            />
        );
    }

    return (
        <Split className="pf-u-align-items-center">
            <SplitItem isFilled>
                <Split className="pf-u-flex-wrap">
                    {tileResources
                        .filter((tileResource) => typeof data[dataKey[tileResource]] === 'number')
                        .map((tileResource) => (
                            <SummaryCount
                                key={tileResource}
                                count={data[dataKey[tileResource]]}
                                href={tileLinks[tileResource]}
                                noun={tileNouns[tileResource]}
                            />
                        ))}
                </Split>
            </SplitItem>
            <div className="pf-u-color-200 pf-u-font-size-sm pf-u-mr-md pf-u-mr-lg-on-lg">
                {`Last updated ${dateFormatter.format(lastUpdate)} at ${timeFormatter.format(
                    lastUpdate
                )}`}
            </div>
        </Split>
    );
}

export default SummaryCounts;
