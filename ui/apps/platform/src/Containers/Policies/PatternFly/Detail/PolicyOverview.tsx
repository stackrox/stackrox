import React, { ReactElement } from 'react';
import { Badge, DescriptionList, Flex, FlexItem, Title } from '@patternfly/react-core';

import DescriptionListItem from 'Components/DescriptionListItem';
import { Policy } from 'types/policy.proto';

import {
    formatCategories,
    formatEventSource,
    formatLifecycleStages,
    formatResponse,
    formatType,
    getEnforcementLifecycleStages,
} from '../policies.utils';
import PolicySeverityLabel from '../PolicySeverityLabel';

type PolicyOverviewProps = {
    policy: Policy;
};

function PolicyOverview({ policy }: PolicyOverviewProps): ReactElement {
    const {
        categories,
        description,
        enforcementActions,
        eventSource,
        exclusions,
        isDefault,
        lifecycleStages,
        notifiers,
        rationale,
        remediation,
        scope,
        severity,
    } = policy;
    const enforcementLifecycleStages = getEnforcementLifecycleStages(
        lifecycleStages,
        enforcementActions
    );

    return (
        <>
            <DescriptionList columnModifier={{ lg: '3Col' }}>
                <DescriptionListItem
                    term="Severity"
                    desc={<PolicySeverityLabel severity={severity} />}
                />
                <DescriptionListItem term="Categories" desc={formatCategories(categories)} />
                <DescriptionListItem term="Type" desc={formatType(isDefault)} />
                <DescriptionListItem term="Description" desc={description} />
                <DescriptionListItem term="Rationale" desc={rationale} />
                <DescriptionListItem term="Guidance" desc={remediation} />
            </DescriptionList>
            <Title headingLevel="h3" className="pf-u-pt-md pf-u-pb-sm">
                Behavior
            </Title>
            <DescriptionList isCompact isHorizontal>
                <DescriptionListItem
                    term="Lifecycle stages"
                    desc={formatLifecycleStages(lifecycleStages)}
                />
                <DescriptionListItem term="Event source" desc={formatEventSource(eventSource)} />
                <DescriptionListItem
                    term="Response"
                    desc={formatResponse(enforcementLifecycleStages)}
                />
                {enforcementLifecycleStages.length !== 0 && (
                    <DescriptionListItem
                        term="Enforcement"
                        desc={formatLifecycleStages(enforcementLifecycleStages)}
                    />
                )}
            </DescriptionList>
            <Flex className="pf-u-pt-md">
                <FlexItem>
                    <Title headingLevel="h3">Notifiers</Title>
                </FlexItem>
                <FlexItem>
                    <Badge isRead>{notifiers.length}</Badge>
                </FlexItem>
            </Flex>
            TODO
            <Flex className="pf-u-pt-md">
                <FlexItem>
                    <Title headingLevel="h3">Scope inclusions</Title>
                </FlexItem>
                <FlexItem>
                    <Badge isRead>{scope.length}</Badge>
                </FlexItem>
            </Flex>
            TODO
            <Flex className="pf-u-pt-md">
                <FlexItem>
                    <Title headingLevel="h3">Scope exclusions</Title>
                </FlexItem>
                <FlexItem>
                    <Badge isRead>{exclusions.length}</Badge>
                </FlexItem>
            </Flex>
            TODO
        </>
    );
}

export default PolicyOverview;
