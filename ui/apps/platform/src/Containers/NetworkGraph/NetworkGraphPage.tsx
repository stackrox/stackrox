import React, { useEffect, useState } from 'react';
import { useParams, useHistory } from 'react-router-dom';
import { PageSection, Title, Flex, FlexItem } from '@patternfly/react-core';
import { Model } from '@patternfly/react-topology';

import { fetchNetworkFlowGraph } from 'services/NetworkService';
import { fetchClustersAsArray, Cluster } from 'services/ClustersService';
import { networkBasePathPF } from 'routePaths';

import PageTitle from 'Components/PageTitle';
import NetworkGraph from './NetworkGraph';
import { transformData, graphModel } from './utils';

import './NetworkGraphPage.css';

function NetworkGraphPage() {
    const history = useHistory();
    const { detailType, detailId } = useParams();
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

    function onSelectNode(type: string, id: string) {
        // if found, and it's not the logical grouping of all external sources, then trigger URL update
        if (id !== 'EXTERNAL') {
            history.push(`${networkBasePathPF}/${type}/${id}`);
        } else {
            // otherwise, return to the graph-only state
            history.push(`${networkBasePathPF}`);
        }
    }

    function closeSidebar() {
        history.push(`${networkBasePathPF}`);
    }

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
                <NetworkGraph
                    detailType={detailType}
                    detailId={detailId}
                    model={model}
                    closeSidebar={closeSidebar}
                    onSelectNode={onSelectNode}
                />
            </PageSection>
        </>
    );
}

export default NetworkGraphPage;
