import { useEffect, useState } from 'react';
import type { ReactElement } from 'react';
import Raven from 'raven-js';
import { Alert, Skeleton, Split, SplitItem } from '@patternfly/react-core';

import {
    clustersBasePath,
    configManagementPath,
    urlEntityListTypes,
    violationsFullViewPath,
    vulnerabilitiesAllImagesPath,
} from 'routePaths';
import { resourceTypes } from 'constants/entityTypes';
import { getDateTime } from 'utils/dateUtils';
import { generatePathWithQuery } from 'utils/searchUtils';

import useSummaryCounts from './hooks/useSummaryCounts';
import type { SummaryCountsResponse, TileResource } from './hooks/useSummaryCounts';
import SummaryCount from './SummaryCount';

const tileResources = ['Cluster', 'Node', 'Alert', 'Deployment', 'Image', 'Secret'] as const;

const dataKey: Record<TileResource, keyof SummaryCountsResponse> = {
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
            vulnerabilitiesAllImagesPath,
            {},
            { customParams: { entityTab: 'Image' } }
        ),
        Secret: `${configManagementPath}/${urlEntityListTypes[resourceTypes.SECRET]}`,
    };

    const [lastUpdate, setLastUpdate] = useState<Date>(new Date());
    const { isLoading, error, data } = useSummaryCounts(hasReadAccessForResource);

    useEffect(() => {
        if (data && !isLoading) {
            setLastUpdate(new Date());
        }
    }, [data, isLoading]);

    if (isLoading) {
        return (
            <Skeleton
                height="32px"
                className="pf-v6-u-m-md"
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
        <Split className="pf-v6-u-align-items-center">
            <SplitItem isFilled>
                <Split className="pf-v6-u-flex-wrap">
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
                                    count={data[dataKey[tileResource]] as number}
                                    href={tileLinks[tileResource]}
                                    noun={tileNouns[tileResource]}
                                    tooltip={tooltip}
                                />
                            );
                        })}
                </Split>
            </SplitItem>
            <div className="pf-v6-u-color-200 pf-v6-u-font-size-sm pf-v6-u-mr-md pf-v6-u-mr-lg-on-lg">
                {`Last updated ${getDateTime(lastUpdate)}`}
            </div>
        </Split>
    );
}

export default SummaryCounts;
