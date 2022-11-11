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

import { fetchNetworkFlowGraph } from 'services/NetworkService';
import { fetchClustersAsArray, Cluster } from 'services/ClustersService';

import PageTitle from 'Components/PageTitle';
import EdgeStateSelect, { EdgeState } from './EdgeStateSelect';
import NetworkGraph from './NetworkGraph';
import { transformData, graphModel } from './utils/modelUtils';

import './NetworkGraphPage.css';

const emptyModel = {
    graph: graphModel,
};

function NetworkGraphPage() {
    const [edgeState, setEdgeState] = useState<EdgeState>('active');
    const [model, setModel] = useState<Model>(emptyModel);
    const [isLoading, setIsLoading] = useState(false);
    const [clusters, setClusters] = useState<Cluster[]>([]);

    useEffect(() => {
        fetchClustersAsArray()
            .then((response) => {
                setClusters(response);
            })
            .catch(() => {
                // TODO
            });
    }, []);

    useEffect(() => {
        if (clusters.length > 0) {
            setIsLoading(true);
            fetchNetworkFlowGraph(clusters[0].id, [])
                .then(({ response }) => {
                    const dataModel = transformData(response.nodes);
                    setModel(dataModel);
                })
                .catch(() => {
                    // TODO
                })
                .finally(() => setIsLoading(false));
        }
    }, [clusters]);

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
            <PageSection className="network-graph" padding={{ default: 'noPadding' }}>
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
                <Divider component="div" />
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
