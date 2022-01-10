import React, { ReactElement } from 'react';
import {
    Card,
    CardBody,
    DescriptionList,
    Grid,
    GridItem,
    List,
    ListItem,
    Title,
} from '@patternfly/react-core';

import DescriptionListItem from 'Components/DescriptionListItem';
import { Cluster } from 'types/cluster.proto';
import { NotifierIntegration } from 'types/notifier.proto';
import { Policy } from 'types/policy.proto';

import {
    formatCategories,
    formatEventSource,
    formatLifecycleStages,
    formatResponse,
    formatType,
    getEnforcementLifecycleStages,
    getExcludedDeployments,
    getExcludedImageNames,
} from '../policies.utils';
import PolicySeverityLabel from '../PolicySeverityLabel';

import ExcludedDeployment from './ExcludedDeployment';
import Notifier from './Notifier';
import Restriction from './Restriction';

type PolicyOverviewProps = {
    clusters: Cluster[];
    notifiers: NotifierIntegration[];
    policy: Policy;
};

function PolicyOverview({ clusters, notifiers, policy }: PolicyOverviewProps): ReactElement {
    const {
        categories,
        description,
        enforcementActions,
        eventSource,
        exclusions,
        isDefault,
        lifecycleStages,
        notifiers: notifierIds,
        rationale,
        remediation,
        scope,
        severity,
    } = policy;
    const enforcementLifecycleStages = getEnforcementLifecycleStages(
        lifecycleStages,
        enforcementActions
    );
    const excludedDeployments = getExcludedDeployments(exclusions);
    const excludedImageNames = getExcludedImageNames(exclusions);

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
            {scope.length !== 0 && (
                <>
                    <Title headingLevel="h3" className="pf-u-pt-md pf-u-pb-sm">
                        Restrict to scopes
                    </Title>
                    <Grid hasGutter>
                        {scope.map((restriction, index) => (
                            // eslint-disable-next-line react/no-array-index-key
                            <GridItem key={index} span={4}>
                                <Card isFlat>
                                    <CardBody>
                                        <Restriction
                                            clusters={clusters}
                                            restriction={restriction}
                                        />
                                    </CardBody>
                                </Card>
                            </GridItem>
                        ))}
                    </Grid>
                </>
            )}
            {excludedDeployments.length !== 0 && (
                <>
                    <Title headingLevel="h3" className="pf-u-pt-md pf-u-pb-sm">
                        Excluded deployments
                    </Title>
                    <Grid hasGutter>
                        {excludedDeployments.map((excludedDeployment, index) => (
                            // eslint-disable-next-line react/no-array-index-key
                            <GridItem key={index} span={4}>
                                <Card isFlat>
                                    <CardBody>
                                        <ExcludedDeployment
                                            clusters={clusters}
                                            excludedDeployment={excludedDeployment}
                                        />
                                    </CardBody>
                                </Card>
                            </GridItem>
                        ))}
                    </Grid>
                </>
            )}
            {excludedImageNames.length !== 0 && (
                <>
                    <Title headingLevel="h3" className="pf-u-pt-md pf-u-pb-sm">
                        Excluded images
                    </Title>
                    <List isPlain>
                        {excludedImageNames.map((name) => (
                            <ListItem key={name}>{name}</ListItem>
                        ))}
                    </List>
                </>
            )}
        </>
    );
}

export default PolicyOverview;
