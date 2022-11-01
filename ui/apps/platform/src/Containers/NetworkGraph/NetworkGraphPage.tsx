import React, { useEffect, useState } from 'react';
import { PageSection, Title, Flex, FlexItem } from '@patternfly/react-core';
import { Model } from '@patternfly/react-topology';

import { fetchNetworkFlowGraph } from 'services/NetworkService';
import { fetchClustersAsArray, Cluster } from 'services/ClustersService';

import PageTitle from 'Components/PageTitle';
import NetworkGraph from './NetworkGraph';
import { transformData, graphModel } from './utils';

import './NetworkGraphPage.css';

function NetworkGraphPage() {
    const [model, setModel] = useState<Model>({
        graph: graphModel,
    });
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
            fetchNetworkFlowGraph(clusters[0].id, [])
                .then(({ response }) => {
                    const dataModel = transformData(response.nodes);
                    setModel(dataModel);
                })
                .catch(() => {
                    // TODO
                });
        }
    }, [clusters]);

    console.log('NetworkGraphPage');

    return (
        <>
            <PageTitle title="Network Graph" />
            <PageSection variant="light">
                <Flex alignItems={{ default: 'alignItemsCenter' }}>
                    <FlexItem flex={{ default: 'flex_1' }}>
                        <Title headingLevel="h1">Network Graph</Title>
                    </FlexItem>
                </Flex>
            </PageSection>
            <PageSection className="network-graph no-padding">
                <NetworkGraph model={model} />
            </PageSection>
        </>
    );
}

export default NetworkGraphPage;
