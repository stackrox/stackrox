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
    ToolbarGroup,
    ToolbarItem,
} from '@patternfly/react-core';
import { Model } from '@patternfly/react-topology';

import { fetchNetworkFlowGraph } from 'services/NetworkService';
import { fetchClustersAsArray, Cluster } from 'services/ClustersService';

import PageTitle from 'Components/PageTitle';
import FlowsSelect, { FlowsState } from './FlowsSelect';
import NetworkGraph from './NetworkGraph';
import { transformData, graphModel } from './utils';

import './NetworkGraphPage.css';

const emptyModel = {
    graph: graphModel,
};

function NetworkGraphPage() {
    const [flowsState, setFlowsState] = useState<FlowsState>('active');
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
                <Toolbar
                    data-testid="network-graph-toolbar"
                    className="theme-light pf-u-px-md pf-u-px-lg-on-xl pf-u-py-sm"
                >
                    <ToolbarGroup spacer={{ default: 'spacerNone' }}>
                        <ToolbarItem>
                            <FlowsSelect flowsState={flowsState} setFlowsState={setFlowsState} />
                        </ToolbarItem>
                    </ToolbarGroup>
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
