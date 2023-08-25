import React, { useState } from 'react';
import { gql, useQuery } from '@apollo/client';
import Raven from 'raven-js';
import pluralize from 'pluralize';
import {
    Alert,
    Button,
    ButtonVariant,
    Skeleton,
    Split,
    SplitItem,
    Stack,
} from '@patternfly/react-core';

import {
    clustersBasePath,
    configManagementPath,
    urlEntityListTypes,
    violationsBasePath,
    vulnManagementImagesPath,
} from 'routePaths';
import { resourceTypes } from 'constants/entityTypes';
import LinkShim from 'Components/PatternFly/LinkShim';

export type SummaryCountsResponse = {
    clusterCount: number;
    nodeCount: number;
    violationCount: number;
    deploymentCount: number;
    imageCount: number;
    secretCount: number;
};

export const SUMMARY_COUNTS = gql`
    query summary_counts {
        clusterCount
        nodeCount
        violationCount
        deploymentCount
        imageCount
        secretCount
    }
`;

const tileEntityTypes = ['Cluster', 'Node', 'Violation', 'Deployment', 'Image', 'Secret'] as const;
type TileEntity = (typeof tileEntityTypes)[number];

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

function SummaryCounts() {
    const [lastUpdate, setLastUpdate] = useState<Date>(new Date());
    const { loading, error, data } = useQuery<SummaryCountsResponse>(SUMMARY_COUNTS, {
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

    const tileData: Record<TileEntity, number> = {
        Cluster: data.clusterCount,
        Node: data.nodeCount,
        Violation: data.violationCount,
        Deployment: data.deploymentCount,
        Image: data.imageCount,
        Secret: data.secretCount,
    };

    return (
        <Split className="pf-u-align-items-center">
            <SplitItem isFilled>
                <Split className="pf-u-flex-wrap">
                    {tileEntityTypes.map((tileEntity) => (
                        <Button
                            key={tileEntity}
                            variant={ButtonVariant.link}
                            component={LinkShim}
                            href={tileLinks[tileEntity]}
                        >
                            <Stack className="pf-u-px-xs pf-u-px-sm-on-xl pf-u-align-items-center">
                                <span className="pf-u-font-size-lg-on-md pf-u-font-size-sm pf-u-font-weight-bold">
                                    {tileData[tileEntity]}
                                </span>
                                <span className="pf-u-font-size-md-on-md pf-u-font-size-xs">
                                    {pluralize(tileEntity, tileData[tileEntity])}
                                </span>
                            </Stack>
                        </Button>
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
