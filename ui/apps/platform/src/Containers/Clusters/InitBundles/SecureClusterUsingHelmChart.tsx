import React, { ReactElement } from 'react';
import { Flex, List, ListItem, Title } from '@patternfly/react-core';

export type SecureClusterUsingHelmChartProps = {
    headingLevel: 'h2' | 'h3';
};

function SecureClusterUsingHelmChart({
    headingLevel,
}: SecureClusterUsingHelmChartProps): ReactElement {
    return (
        <Flex direction={{ default: 'column' }}>
            <Title headingLevel={headingLevel}>
                Secure a cluster using Helm chart installation method
            </Title>
            <List component="ol">
                <ListItem>Do something.</ListItem>
                <ListItem>Do something.</ListItem>
                <ListItem>Do something.</ListItem>
            </List>
        </Flex>
    );
}

export default SecureClusterUsingHelmChart;
