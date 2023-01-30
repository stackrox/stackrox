import React from 'react';
import { Flex, FlexItem, Title } from '@patternfly/react-core';
import { PficonNetworkRangeIcon, BuilderImageIcon } from '@patternfly/react-icons';

import DescriptionListItem from 'Components/DescriptionListItem';
import DescriptionListCompact from 'Components/DescriptionListCompact';

import { ReactComponent as BothPolicyRules } from 'images/network-graph/both-policy-rules.svg';
import { ReactComponent as EgressOnly } from 'images/network-graph/egress-only.svg';
import { ReactComponent as IngressOnly } from 'images/network-graph/ingress-only.svg';
import { ReactComponent as NoPolicyRules } from 'images/network-graph/no-policy-rules.svg';

function LegendContent() {
    return (
        <>
            <Title headingLevel="h3" className="pf-u-screen-reader">
                Legend
            </Title>
            <Flex direction={{ default: 'column' }}>
                <FlexItem>
                    <Title headingLevel="h5" className="pf-u-pb-sm">
                        Node types
                    </Title>
                    <DescriptionListCompact isHorizontal termWidth="20px" className="pf-u-pl-md">
                        <DescriptionListItem term={<BuilderImageIcon />} desc="Deployment" />
                        <DescriptionListItem
                            term={<PficonNetworkRangeIcon />}
                            desc="External CIDR block"
                        />
                    </DescriptionListCompact>
                </FlexItem>
                <FlexItem>
                    <Title headingLevel="h5" className="pf-u-pb-sm">
                        Deployment badges
                    </Title>
                    <DescriptionListCompact isHorizontal termWidth="20px" className="pf-u-pl-md">
                        <DescriptionListItem
                            term={<PficonNetworkRangeIcon className="pf-u-ml-xs" />}
                            desc="Connected to external entities"
                            groupClassName="pf-u-align-items-center"
                        />
                        <DescriptionListItem
                            term={<BothPolicyRules width="22px" height="22px" />}
                            desc="Isolated by network policy rules"
                            groupClassName="pf-u-align-items-center"
                        />
                        <DescriptionListItem
                            term={<NoPolicyRules width="22px" height="22px" />}
                            desc="All traffic allowed (No network policies)"
                            groupClassName="pf-u-align-items-center"
                        />
                        <DescriptionListItem
                            term={<EgressOnly width="22px" height="22px" />}
                            desc="Only has an egress network policy"
                            groupClassName="pf-u-align-items-center"
                        />
                        <DescriptionListItem
                            term={<IngressOnly width="22px" height="22px" />}
                            desc="Only has an ingress network policy"
                            groupClassName="pf-u-align-items-center"
                        />
                    </DescriptionListCompact>
                </FlexItem>
            </Flex>
        </>
    );
}

export default LegendContent;
