import React, { useEffect, useState } from 'react';
import { PageSection, Title, Flex, FlexItem, Bullseye, Spinner } from '@patternfly/react-core';
import { Model } from '@patternfly/react-topology';

import { fetchNetworkFlowGraph } from 'services/NetworkService';
import { fetchClustersAsArray, Cluster } from 'services/ClustersService';

import PageTitle from 'Components/PageTitle';
import NetworkGraph from './NetworkGraph';
import { transformData, graphModel } from './utils';

import './NetworkGraphPage.css';

const emptyModel = {
    graph: graphModel,
};

function NetworkGraphPage() {
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
                </Flex>
            </PageSection>
            <PageSection className="network-graph no-padding">
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
