import React from 'react';
import { Flex, FlexItem, Title } from '@patternfly/react-core';
import { PficonNetworkRangeIcon, BuilderImageIcon } from '@patternfly/react-icons';

import { ReactComponent as BothPolicyRules } from 'images/network-graph/both-policy-rules.svg';
import { ReactComponent as EgressOnly } from 'images/network-graph/egress-only.svg';
import { ReactComponent as IngressOnly } from 'images/network-graph/ingress-only.svg';
import { ReactComponent as NoPolicyRules } from 'images/network-graph/no-policy-rules.svg';

function LegendContent() {
    return (
        <Flex>
            <FlexItem>
                <Title headingLevel="h4">Node types</Title>
                <Flex>
                    <FlexItem>
                        <BuilderImageIcon />
                        <div>Deployment</div>
                    </FlexItem>
                    <FlexItem>
                        <PficonNetworkRangeIcon />
                        <div>External CIDR block</div>
                    </FlexItem>
                </Flex>
            </FlexItem>
            <FlexItem>
                <Title headingLevel="h4">Deployment badges</Title>
                <Flex>
                    <FlexItem>
                        <BothPolicyRules />
                        <div>Isolated by network poilcy rules</div>
                    </FlexItem>
                    <FlexItem>
                        <NoPolicyRules />
                        <div>All traffic allowed (No network policies)</div>
                    </FlexItem>
                    <FlexItem>
                        <EgressOnly />
                        <div>Only has an egress network policy</div>
                    </FlexItem>
                    <FlexItem>
                        <IngressOnly />
                        <div>Only has an ingress network policy</div>
                    </FlexItem>
                </Flex>
            </FlexItem>
        </Flex>
    );
}

export default LegendContent;
