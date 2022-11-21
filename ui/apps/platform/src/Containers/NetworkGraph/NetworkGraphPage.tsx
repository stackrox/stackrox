import React, { useEffect, useState } from 'react';
import {
    PageSection,
    Title,
    Flex,
    FlexItem,
    Bullseye,
    Spinner,
    Button,
    Divider,
    Toolbar,
    ToolbarContent,
    ToolbarGroup,
    ToolbarItem,
} from '@patternfly/react-core';
import { Model } from '@patternfly/react-topology';

import useFetchClusters from 'hooks/useFetchClusters';
import useURLSearch from 'hooks/useURLSearch';
import { SearchFilter } from 'types/search';
import { fetchNetworkFlowGraph } from 'services/NetworkService';
import { getQueryString } from 'utils/queryStringUtils';
import timeWindowToDate from 'utils/timeWindows';

import PageTitle from 'Components/PageTitle';
import EdgeStateSelect, { EdgeState } from './EdgeStateSelect';
import NetworkGraph from './NetworkGraph';
import { transformData, graphModel } from './utils/modelUtils';

import './NetworkGraphPage.css';

const emptyModel = {
    graph: graphModel,
};

// TODO: get real time window from user input
const timeWindow = 'Past hours';

// TODO: refactor to another file and import
function getScopeHierarchyFromSearch(searchFilter: SearchFilter) {
    const hierarchy: { cluster: string | undefined; namespaces: string[]; deployments: string[] } =
        {
            cluster: undefined,
            namespaces: [],
            deployments: [],
        };

    if (!searchFilter.Cluster || Array.isArray(searchFilter.Cluster)) {
        return hierarchy;
    }
    hierarchy.cluster = searchFilter.Cluster;

    if (searchFilter.Namespace) {
        hierarchy.namespaces = Array.isArray(searchFilter.Namespace)
            ? searchFilter.Namespace
            : [searchFilter.Namespace];

        if (searchFilter.Deployment) {
            hierarchy.deployments = Array.isArray(searchFilter.Deployment)
                ? searchFilter.Deployment
                : [searchFilter.Deployment];
        }
    }
    return hierarchy;
}

function NetworkGraphPage() {
    const [edgeState, setEdgeState] = useState<EdgeState>('active');
    const [model, setModel] = useState<Model>(emptyModel);
    const [isLoading, setIsLoading] = useState(false);
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    const { searchFilter, setSearchFilter } = useURLSearch();

    const {
        cluster: clusterFromUrl,
        namespaces: namespacesFromUrl,
        // eslint-disable-next-line @typescript-eslint/no-unused-vars
        deployments: deploymentsFromUrl,
    } = getScopeHierarchyFromSearch(searchFilter);

    const { clusters } = useFetchClusters();

    useEffect(() => {
        if (clusters.length > 0 && clusterFromUrl) {
            const selectedClusterId = clusters.find((cl) => cl.name === clusterFromUrl)?.id;
            if (selectedClusterId) {
                setIsLoading(true);

                const remainingQuery = { ...searchFilter };
                delete remainingQuery.Cluster;
                delete remainingQuery.Namespace;

                const queryToUse = getQueryString(remainingQuery);

                fetchNetworkFlowGraph(
                    selectedClusterId,
                    namespacesFromUrl,
                    queryToUse,
                    timeWindowToDate(timeWindow) || undefined
                )
                    .then(({ response }) => {
                        const dataModel = transformData(response.nodes);
                        setModel(dataModel);
                    })
                    .catch(() => {
                        // TODO
                    })
                    .finally(() => setIsLoading(false));
            }
        }
    }, [clusters, clusterFromUrl]);

    return (
        <>
            <PageTitle title="Network Graph" />
            <PageSection variant="light">
                <Flex alignItems={{ default: 'alignItemsCenter' }}>
                    <FlexItem flex={{ default: 'flex_1' }}>
                        <Title headingLevel="h1">Network Graph</Title>
                    </FlexItem>
                    <Button variant="secondary">Manage CIDR blocks</Button>
                    <Button variant="secondary">Simulate network policy</Button>
                </Flex>
            </PageSection>
            <Divider component="div" />
            <PageSection variant="light" padding={{ default: 'noPadding' }}>
                <Toolbar data-testid="network-graph-toolbar">
                    <ToolbarContent>
                        <ToolbarGroup variant="filter-group">
                            <ToolbarItem>
                                <EdgeStateSelect
                                    edgeState={edgeState}
                                    setEdgeState={setEdgeState}
                                />
                            </ToolbarItem>
                            <ToolbarItem>in the past hour</ToolbarItem>
                        </ToolbarGroup>
                        <ToolbarGroup>
                            <ToolbarItem>Add one or more deployment filters</ToolbarItem>
                            <ToolbarItem>Display options</ToolbarItem>
                        </ToolbarGroup>
                        <ToolbarGroup alignment={{ default: 'alignRight' }}>
                            <Divider component="div" isVertical />
                            <ToolbarItem>Last updated at 12:34PM</ToolbarItem>
                        </ToolbarGroup>
                    </ToolbarContent>
                </Toolbar>
            </PageSection>
            <Divider component="div" />
            <PageSection className="network-graph" padding={{ default: 'noPadding' }}>
                {model.nodes && <NetworkGraph model={model} />}
                {isLoading && (
                    <Bullseye>
                        <Spinner isSVG />
                    </Bullseye>
                )}
            </PageSection>
        </>
    );
}

export default NetworkGraphPage;
