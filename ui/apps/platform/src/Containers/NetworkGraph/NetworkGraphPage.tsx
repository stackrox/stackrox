import React, { useEffect } from 'react';
import { useParams } from 'react-router-dom';
import { PageSection, Title, Flex, FlexItem } from '@patternfly/react-core';

import { fetchNetworkFlowGraph } from 'services/NetworkService';
import PageTitle from 'Components/PageTitle';
import NetworkGraph from './NetworkGraph';

import './NetworkGraphPage.css';

function NetworkGraphPage() {
    const { detailType, detailId } = useParams();

    // useEffect(() => {
    //     fetchNetworkFlowGraph()
    // },[]);

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
                <NetworkGraph detailType={detailType} detailId={detailId} />
            </PageSection>
        </>
    );
}

export default NetworkGraphPage;
