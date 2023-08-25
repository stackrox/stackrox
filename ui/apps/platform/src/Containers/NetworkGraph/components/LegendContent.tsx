import React from 'react';
import { Flex, FlexItem, Title } from '@patternfly/react-core';
import {
    PficonNetworkRangeIcon,
    BuilderImageIcon,
    ExclamationCircleIcon,
} from '@patternfly/react-icons';

import DescriptionListItem from 'Components/DescriptionListItem';
import DescriptionListCompact from 'Components/DescriptionListCompact';

import { ReactComponent as BothPolicyRules } from 'images/network-graph/both-policy-rules.svg';
import { ReactComponent as EgressOnly } from 'images/network-graph/egress-only.svg';
import { ReactComponent as IngressOnly } from 'images/network-graph/ingress-only.svg';
import { ReactComponent as NoPolicyRules } from 'images/network-graph/no-policy-rules.svg';
import { ReactComponent as RelatedNSBorder } from 'images/network-graph/related-ns-border.svg';
import { ReactComponent as RelatedEntity } from 'images/network-graph/related-entity.svg';
import { ReactComponent as FilteredEntity } from 'images/network-graph/filtered-entity.svg';

function LegendContent() {
    return (
        <>
            <Title headingLevel="h3" className="pf-u-screen-reader" data-testid="legend-title">
                Legend
            </Title>
            <Flex direction={{ default: 'column' }}>
                <FlexItem>
                    <Title headingLevel="h4" className="pf-u-pb-sm" data-testid="node-types-title">
                        Node types
                    </Title>
                    <DescriptionListCompact isHorizontal termWidth="20px" className="pf-u-pl-md">
                        <DescriptionListItem
                            term={<BuilderImageIcon />}
                            desc="Deployment"
                            groupClassName="pf-u-align-items-center"
                        />
                        <DescriptionListItem
                            term={<PficonNetworkRangeIcon />}
                            desc="External CIDR block"
                            groupClassName="pf-u-align-items-center"
                        />
                    </DescriptionListCompact>
                </FlexItem>
                <FlexItem>
                    <Title
                        headingLevel="h4"
                        className="pf-u-pb-sm"
                        data-testid="namespace-types-title"
                    >
                        Namespace types
                    </Title>
                    <DescriptionListCompact isHorizontal termWidth="20px" className="pf-u-pl-md">
                        <DescriptionListItem
                            term={<FilteredEntity width="20px" height="20px" />}
                            desc="Filtered namespace"
                            groupClassName="pf-u-align-items-center"
                        />
                        <DescriptionListItem
                            term={<RelatedEntity width="18px" height="18px" />}
                            desc="Related namespace"
                            groupClassName="pf-u-align-items-center"
                        />
                        <DescriptionListItem
                            term={<RelatedNSBorder />}
                            desc="Related namespace grouping"
                            groupClassName="pf-u-align-items-center"
                        />
                    </DescriptionListCompact>
                </FlexItem>
                <FlexItem>
                    <Title
                        headingLevel="h4"
                        className="pf-u-pb-sm"
                        data-testid="deployment-types-title"
                    >
                        Deployment types
                    </Title>
                    <DescriptionListCompact isHorizontal termWidth="24px" className="pf-u-pl-md">
                        <DescriptionListItem
                            term={<FilteredEntity width="20px" height="20px" />}
                            desc="Filtered deployment"
                            groupClassName="pf-u-align-items-center"
                        />
                    </DescriptionListCompact>
                </FlexItem>
                <FlexItem>
                    <Title
                        headingLevel="h4"
                        className="pf-u-pb-sm"
                        data-testid="deployment-badges-title"
                    >
                        Deployment badges
                    </Title>
                    <DescriptionListCompact isHorizontal termWidth="20px" className="pf-u-pl-md">
                        <DescriptionListItem
                            term={
                                <ExclamationCircleIcon className="pf-u-ml-xs pf-u-danger-color-100" />
                            }
                            desc="Anomalous traffic detected"
                            groupClassName="pf-u-align-items-center"
                        />
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
                            term={<IngressOnly width="22px" height="22px" />}
                            desc="Only has an ingress network policy"
                            groupClassName="pf-u-align-items-center"
                        />
                        <DescriptionListItem
                            term={<EgressOnly width="22px" height="22px" />}
                            desc="Only has an egress network policy"
                            groupClassName="pf-u-align-items-center"
                        />
                    </DescriptionListCompact>
                </FlexItem>
            </Flex>
        </>
    );
}

export default LegendContent;
