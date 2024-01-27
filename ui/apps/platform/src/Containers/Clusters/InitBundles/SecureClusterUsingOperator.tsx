import React, { ReactElement } from 'react';
import { Flex, List, ListItem, Title } from '@patternfly/react-core';

export type SecureClusterUsingOperatorProps = {
    headingLevel: 'h2' | 'h3';
};

function SecureClusterUsingOperator({
    headingLevel,
}: SecureClusterUsingOperatorProps): ReactElement {
    return (
        <Flex direction={{ default: 'column' }}>
            <Title headingLevel={headingLevel}>
                Secure a cluster using Operator installation method
            </Title>
            <List component="ol">
                <ListItem>Do something.</ListItem>
                <ListItem>Do something.</ListItem>
                <ListItem>Do something.</ListItem>
            </List>
        </Flex>
    );
}

export default SecureClusterUsingOperator;
