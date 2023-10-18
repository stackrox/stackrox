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

const tileEntityTypes = ['Cluster', 'Node', 'Violation', 'Deployment', 'Image', 'Secret'] as const;
type TileEntity = (typeof tileEntityTypes)[number];

const dataKey: Record<TileEntity, string> = {
    Cluster: 'clusterCount',
    Node: 'nodeCount',
    Violation: 'violationCount',
    Deployment: 'deploymentCount',
    Image: 'imageCount',
    Secret: 'secretCount',
};

const tileLinks: Record<TileEntity, string> = {
    Cluster: clustersBasePath,
    Node: `${configManagementPath}/${urlEntityListTypes[resourceTypes.NODE]}`,
    Violation: violationsBasePath,
    Deployment: `${configManagementPath}/${urlEntityListTypes[resourceTypes.DEPLOYMENT]}`,
    Image: vulnManagementImagesPath,
    Secret: `${configManagementPath}/${urlEntityListTypes[resourceTypes.SECRET]}`,
};

const locale = window.navigator.language ?? 'en-US';
const dateFormatter = new Intl.DateTimeFormat(locale);
const timeFormatter = new Intl.DateTimeFormat(locale, { hour: 'numeric', minute: 'numeric' });

export type SummaryCountsProps = Record<TileEntity, boolean>;

function SummaryCounts(hasReadAccessForTileEntity: SummaryCountsProps): ReactElement {
    const query = gql`
        query summary_counts {
            ${tileEntityTypes
                .filter((tileEntity) => hasReadAccessForTileEntity[tileEntity])
                .map((tileEntity) => dataKey[tileEntity])
                .join('\n')}
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
                className="pf-u-m-md"
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
            />
        );
    }

    return (
        <Split className="pf-u-align-items-center">
            <SplitItem isFilled>
                <Split className="pf-u-flex-wrap">
                    {tileEntityTypes
                        .filter((tileEntity) => typeof data[dataKey[tileEntity]] === 'number')
                        .map((tileEntity) => (
                            <SummaryCount
                                key={tileEntity}
                                count={data[dataKey[tileEntity]]}
                                href={tileLinks[tileEntity]}
                                noun={tileEntity}
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
