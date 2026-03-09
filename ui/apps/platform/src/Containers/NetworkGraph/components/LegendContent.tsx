import { DescriptionList, Flex, FlexItem, Title } from '@patternfly/react-core';
import {
    BuilderImageIcon,
    ExclamationCircleIcon,
    PficonNetworkRangeIcon,
} from '@patternfly/react-icons';

import DescriptionListItem from 'Components/DescriptionListItem';

import BothPolicyRules from 'images/network-graph/both-policy-rules.svg?react';
import EgressOnly from 'images/network-graph/egress-only.svg?react';
import IngressOnly from 'images/network-graph/ingress-only.svg?react';
import NoPolicyRules from 'images/network-graph/no-policy-rules.svg?react';
import RelatedNSBorder from 'images/network-graph/related-ns-border.svg?react';
import RelatedEntity from 'images/network-graph/related-entity.svg?react';
import FilteredEntity from 'images/network-graph/filtered-entity.svg?react';

const defaultTermWidth = '1rem'; // Matches the default icon width set by .pf-v6-svg

// Needed to restore styles to the spacing we had in PatternFly 5, as the upgrade to PatternFly 6 removed the spacing.
// We may want to evaluate an alternative component as a future follow-up.
const xsRowGapStyle = {
    '--pf-v6-c-description-list--m-compact--RowGap': 'var(--pf-t--global--spacer--xs)',
};

function LegendContent() {
    return (
        <>
            <Title headingLevel="h3" className="pf-v6-u-screen-reader" data-testid="legend-title">
                Legend
            </Title>
            <Flex direction={{ default: 'column' }}>
                <FlexItem>
                    <Title
                        headingLevel="h4"
                        className="pf-v6-u-pb-sm"
                        data-testid="node-types-title"
                    >
                        Node types
                    </Title>
                    <DescriptionList
                        isCompact
                        isHorizontal
                        className="pf-v6-u-pl-md"
                        horizontalTermWidthModifier={{ default: defaultTermWidth }}
                        style={xsRowGapStyle}
                    >
                        <DescriptionListItem
                            term={<BuilderImageIcon />}
                            desc="Deployment"
                            groupClassName="pf-v6-u-align-items-center"
                        />
                        <DescriptionListItem
                            term={<PficonNetworkRangeIcon />}
                            desc="External CIDR block"
                            groupClassName="pf-v6-u-align-items-center"
                        />
                    </DescriptionList>
                </FlexItem>
                <FlexItem>
                    <Title
                        headingLevel="h4"
                        className="pf-v6-u-pb-sm"
                        data-testid="namespace-types-title"
                    >
                        Namespace types
                    </Title>
                    <DescriptionList
                        isCompact
                        isHorizontal
                        className="pf-v6-u-pl-md"
                        horizontalTermWidthModifier={{ default: defaultTermWidth }}
                        style={xsRowGapStyle}
                    >
                        <DescriptionListItem
                            term={<FilteredEntity width="20px" height="20px" />}
                            desc="Filtered namespace"
                            groupClassName="pf-v6-u-align-items-center"
                        />
                        <DescriptionListItem
                            term={<RelatedEntity width="18px" height="18px" />}
                            desc="Related namespace"
                            groupClassName="pf-v6-u-align-items-center"
                        />
                        <DescriptionListItem
                            term={<RelatedNSBorder />}
                            desc="Related namespace grouping"
                            groupClassName="pf-v6-u-align-items-center"
                        />
                    </DescriptionList>
                </FlexItem>
                <FlexItem>
                    <Title
                        headingLevel="h4"
                        className="pf-v6-u-pb-sm"
                        data-testid="deployment-types-title"
                    >
                        Deployment types
                    </Title>
                    <DescriptionList
                        isCompact
                        isHorizontal
                        className="pf-v6-u-pl-md"
                        horizontalTermWidthModifier={{ default: defaultTermWidth }}
                        style={xsRowGapStyle}
                    >
                        <DescriptionListItem
                            term={<FilteredEntity width="20px" height="20px" />}
                            desc="Filtered deployment"
                            groupClassName="pf-v6-u-align-items-center"
                        />
                    </DescriptionList>
                </FlexItem>
                <FlexItem>
                    <Title
                        headingLevel="h4"
                        className="pf-v6-u-pb-sm"
                        data-testid="deployment-badges-title"
                    >
                        Deployment badges
                    </Title>
                    <DescriptionList
                        isCompact
                        isHorizontal
                        className="pf-v6-u-pl-md"
                        horizontalTermWidthModifier={{ default: defaultTermWidth }}
                        style={xsRowGapStyle}
                    >
                        <DescriptionListItem
                            term={
                                <ExclamationCircleIcon className="pf-v6-u-ml-xs pf-v6-u-text-color-status-danger" />
                            }
                            desc="Anomalous traffic detected"
                            groupClassName="pf-v6-u-align-items-center"
                        />
                        <DescriptionListItem
                            term={<PficonNetworkRangeIcon className="pf-v6-u-ml-xs" />}
                            desc="Connected to external entities"
                            groupClassName="pf-v6-u-align-items-center"
                        />
                        <DescriptionListItem
                            term={<BothPolicyRules width="22px" height="22px" />}
                            desc="Isolated by network policy rules"
                            groupClassName="pf-v6-u-align-items-center"
                        />
                        <DescriptionListItem
                            term={<NoPolicyRules width="22px" height="22px" />}
                            desc="All traffic allowed (No network policies)"
                            groupClassName="pf-v6-u-align-items-center"
                        />
                        <DescriptionListItem
                            term={<IngressOnly width="22px" height="22px" />}
                            desc="Only has an ingress network policy"
                            groupClassName="pf-v6-u-align-items-center"
                        />
                        <DescriptionListItem
                            term={<EgressOnly width="22px" height="22px" />}
                            desc="Only has an egress network policy"
                            groupClassName="pf-v6-u-align-items-center"
                        />
                    </DescriptionList>
                </FlexItem>
            </Flex>
        </>
    );
}

export default LegendContent;
