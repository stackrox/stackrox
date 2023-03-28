import React, { ReactElement } from 'react';
import { Text, TextContent, TextVariants } from '@patternfly/react-core';

type FlowsTableHeaderTextProps = {
    type: 'baseline' | 'active' | 'inactive' | 'baseline simulated';
    numFlows: number;
};

function FlowsTableHeaderText({ type, numFlows }: FlowsTableHeaderTextProps): ReactElement {
    return (
        <TextContent>
            <Text component={TextVariants.h3} data-testid="flows-table-header">
                {numFlows} {type} flows
            </Text>
        </TextContent>
    );
}

export default FlowsTableHeaderText;
