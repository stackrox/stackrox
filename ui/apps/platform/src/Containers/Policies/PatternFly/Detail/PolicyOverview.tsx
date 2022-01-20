import React, { ReactElement } from 'react';
import { Card, CardBody, DescriptionList, Grid, GridItem, Title } from '@patternfly/react-core';

import DescriptionListItem from 'Components/DescriptionListItem';
import { NotifierIntegration } from 'types/notifier.proto';
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

import Notifier from './Notifier';

type PolicyOverviewProps = {
    notifiers: NotifierIntegration[];
    policy: Policy;
};

function PolicyOverview({ notifiers, policy }: PolicyOverviewProps): ReactElement {
    const {
        categories,
        description,
        enforcementActions,
        eventSource,
        isDefault,
        lifecycleStages,
        notifiers: notifierIds,
        rationale,
        remediation,
        severity,
    } = policy;
    const enforcementLifecycleStages = getEnforcementLifecycleStages(
        lifecycleStages,
        enforcementActions
    );

    return (
        <>
            <DescriptionList isCompact isHorizontal>
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
            {notifierIds.length !== 0 && (
                <>
                    <Title headingLevel="h3" className="pf-u-pt-md pf-u-pb-sm">
                        Notifiers
                    </Title>
                    <Grid hasGutter>
                        {notifierIds.map((notifierId) => (
                            <GridItem key={notifierId} span={4}>
                                <Card isFlat>
                                    <CardBody>
                                        <Notifier notifierId={notifierId} notifiers={notifiers} />
                                    </CardBody>
                                </Card>
                            </GridItem>
                        ))}
                    </Grid>
                </>
            )}
        </>
    );
}

export default PolicyOverview;
