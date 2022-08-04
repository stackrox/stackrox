import React from 'react';
import { capitalize, Flex, FlexItem, Text, Title } from '@patternfly/react-core';
import { ExclamationCircleIcon } from '@patternfly/react-icons';

import EmptyStateTemplate from 'Components/PatternFly/EmptyStateTemplate';

type GraphLoadErrorStateProps = {
    /** The direct error message returned from the server */
    error: string;
    /** A user friendly message to describe how to resolve the error */
    userMessage: string;
};

function GraphLoadErrorState({ error, userMessage }: GraphLoadErrorStateProps) {
    return (
        <Flex className="pf-u-flex-grow-1 pf-u-pt-2xl">
            <FlexItem className="pf-u-color-100" grow={{ default: 'grow' }}>
                <EmptyStateTemplate
                    headingLevel="h2"
                    title="An error has prevented the Network Graph from loading."
                    icon={ExclamationCircleIcon}
                >
                    <Title headingLevel="h3">{capitalize(error)}</Title>
                    <Text className="pf-u-mt-lg">{userMessage}</Text>
                </EmptyStateTemplate>
            </FlexItem>
        </Flex>
    );
}

export default GraphLoadErrorState;
