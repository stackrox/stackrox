import React, { ReactElement, useState } from 'react';
import { gql, useQuery } from '@apollo/client';
import Raven from 'raven-js';
import { Alert, Skeleton, Split, SplitItem } from '@patternfly/react-core';

import {
    clustersBasePath,
    configManagementPath,
    urlEntityListTypes,
    violationsFullViewPath,
    vulnerabilitiesWorkloadCvesPath,
} from 'routePaths';
import { resourceTypes } from 'constants/entityTypes';
import { getDateTime } from 'utils/dateUtils';
import { generatePathWithQuery } from 'utils/searchUtils';

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

const tileNouns: Record<TileResource, string> = {
    Cluster: 'Cluster',
    Node: 'Node',
    Alert: 'Violation',
    Deployment: 'Deployment',
    Image: 'Image',
    Secret: 'Secret',
};

export type SummaryCountsProps = {
    hasReadAccessForResource: Record<TileResource, boolean>;
};

function SummaryCounts({ hasReadAccessForResource }: SummaryCountsProps): ReactElement {
    // According to current minimalist philosophy, ignore that routes might have additional resource requirements.
    const tileLinks: Record<TileResource, string> = {
        Cluster: clustersBasePath,
        Node: `${configManagementPath}/${urlEntityListTypes[resourceTypes.NODE]}`,
        Alert: violationsFullViewPath,
        Deployment: `${configManagementPath}/${urlEntityListTypes[resourceTypes.DEPLOYMENT]}`,
        Image: generatePathWithQuery(
            vulnerabilitiesWorkloadCvesPath,
            {},
            { customParams: { entityTab: 'Image' } }
        ),
        Secret: `${configManagementPath}/${urlEntityListTypes[resourceTypes.SECRET]}`,
    };

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
    const { loading, error, data } = useQuery<SummaryCountsResponse>(query, {
        fetchPolicy: 'network-only',
        onCompleted: () => setLastUpdate(new Date()),
    });

    if (loading) {
        return (
            <Skeleton
                height="32px"
                className="pf-v5-u-m-md"
                screenreaderText="Loading system summary counts"
            />
        );
    }

    if (error || !data) {
        Raven.captureException(error);
        return (
            <Alert
                isInline
                variant="warning"
                title="There was an error loading system summary counts"
                component="p"
            />
        );
    }

    return (
        <Split className="pf-v5-u-align-items-center">
            <SplitItem isFilled>
                <Split className="pf-v5-u-flex-wrap">
                    {tileResources
                        .filter((tileResource) => typeof data[dataKey[tileResource]] === 'number')
                        .map((tileResource) => {
                            const tooltip =
                                tileResource === 'Image'
                                    ? 'Count includes all images, with or without observed CVEs'
                                    : undefined;

                            return (
                                <SummaryCount
                                    key={tileResource}
                                    count={data[dataKey[tileResource]]}
                                    href={tileLinks[tileResource]}
                                    noun={tileNouns[tileResource]}
                                    tooltip={tooltip}
                                />
                            );
                        })}
                </Split>
            </SplitItem>
            <div className="pf-v5-u-color-200 pf-v5-u-font-size-sm pf-v5-u-mr-md pf-v5-u-mr-lg-on-lg">
                {`Last updated ${getDateTime(lastUpdate)}`}
            </div>
        </Split>
    );
}

export default SummaryCounts;
