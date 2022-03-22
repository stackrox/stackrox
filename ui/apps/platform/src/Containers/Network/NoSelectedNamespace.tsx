import React from 'react';
import { Flex, FlexItem } from '@patternfly/react-core';

import EmptyStateTemplate from 'Components/PatternFly/EmptyStateTemplate';
import NamespaceSelect from './Header/NamespaceSelect';

interface NoSelectedNamespaceProps {
    clusterName: string;
}

function NoSelectedNamespace({ clusterName }: NoSelectedNamespaceProps) {
    return (
        <Flex
            direction={{ default: 'row' }}
            flex={{ default: 'flex_1' }}
            alignItems={{ default: 'alignItemsCenter' }}
            className="pf-u-flex-grow-1"
        >
            <FlexItem grow={{ default: 'grow' }}>
                <EmptyStateTemplate
                    headingLevel="h2"
                    title={`Please select at least one namespace from the ${clusterName} cluster`}
                >
                    <NamespaceSelect direction="up" />
                </EmptyStateTemplate>
            </FlexItem>
        </Flex>
    );
}

export default NoSelectedNamespace;
