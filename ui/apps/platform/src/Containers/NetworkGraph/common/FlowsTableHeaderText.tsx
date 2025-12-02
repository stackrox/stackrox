import type { ReactElement } from 'react';
import { Content, ContentVariants } from '@patternfly/react-core';

type FlowsTableHeaderTextProps = {
    type: 'baseline' | 'active' | 'inactive' | 'baseline simulated' | 'total';
    numFlows: number;
};

function FlowsTableHeaderText({ type, numFlows }: FlowsTableHeaderTextProps): ReactElement {
    return (
        <Content>
            <Content component={ContentVariants.h3}>
                {numFlows} {type} flows
            </Content>
        </Content>
    );
}

export default FlowsTableHeaderText;
