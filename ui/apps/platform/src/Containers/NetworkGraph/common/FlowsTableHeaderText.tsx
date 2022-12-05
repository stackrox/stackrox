import React, { ReactElement } from 'react';
import { Text, TextContent, TextVariants } from '@patternfly/react-core';

type FlowsTableHeaderTextProps = {
    type: 'baseline' | 'active' | 'extraneous';
    numFlows: number;
};

function FlowsTableHeaderText({ type, numFlows }: FlowsTableHeaderTextProps): ReactElement {
    return (
        <TextContent>
            <Text component={TextVariants.h3}>
                {numFlows} {type} flows
            </Text>
        </TextContent>
    );
}

export default FlowsTableHeaderText;
