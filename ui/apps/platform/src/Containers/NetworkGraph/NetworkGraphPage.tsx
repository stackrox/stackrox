import React from 'react';
import { PageSection, Title, Flex, FlexItem } from '@patternfly/react-core';

import PageTitle from 'Components/PageTitle';
import NetworkGraph from './NetworkGraph';

import './NetworkGraphPage.css';

function NetworkGraphPage() {
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
                <NetworkGraph />
            </PageSection>
        </>
    );
}

export default NetworkGraphPage;
