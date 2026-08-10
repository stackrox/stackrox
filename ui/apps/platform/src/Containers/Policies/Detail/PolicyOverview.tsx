import type { ReactElement } from 'react';
import {
    Card,
    CardBody,
    CardHeader,
    DescriptionList,
    Divider,
    Grid,
    GridItem,
    Label,
    Title,
} from '@patternfly/react-core';

import DescriptionListItem from 'Components/DescriptionListItem';
import PolicySeverityIconText from 'Components/PatternFly/IconText/PolicySeverityIconText';
import type { NotifierIntegration } from 'types/notifier.proto';
import type { BasePolicy } from 'types/policy.proto';
import MitreAttackVectorsViewContainer from 'Containers/MitreAttackVectors/MitreAttackVectorsViewContainer';

import { formatCategories, getPolicyOriginLabel, getNotifierTypeLabel } from '../policies.utils';

type PolicyOverviewProps = {
    notifiers: NotifierIntegration[];
    policy: BasePolicy;
    isReview?: boolean;
};

function PolicyOverview({
    notifiers,
    policy,
    isReview = false,
}: PolicyOverviewProps): ReactElement {
    const {
        categories,
        description,
        notifiers: notifierIds,
        notifierToCollectionMappings,
        rationale,
        remediation,
        severity,
        name,
    } = policy;

    const hasUnscopedNotifiers = notifierIds?.length > 0;
    const hasScopedNotifiers = notifierToCollectionMappings?.length > 0;
    const hasAnyNotifiers = hasUnscopedNotifiers || hasScopedNotifiers;

    function getNotifierInfo(notifierId: string) {
        const notifier = notifiers.find(({ id }) => id === notifierId);
        const typeLabel = getNotifierTypeLabel(notifier?.type ?? '');
        return { name: notifier?.name ?? notifierId, typeLabel };
    }

    return (
        <Card isFlat>
            {isReview && (
                <CardHeader>
                    <Title headingLevel="h2" size="lg">
                        {name}
                    </Title>
                </CardHeader>
            )}
            <CardBody>
                <DescriptionList isCompact isHorizontal>
                    <DescriptionListItem
                        term="Severity"
                        desc={<PolicySeverityIconText severity={severity} />}
                    />
                    <DescriptionListItem term="Categories" desc={formatCategories(categories)} />
                    <DescriptionListItem term="Origin" desc={getPolicyOriginLabel(policy)} />
                    <DescriptionListItem term="Description" desc={description} />
                    <DescriptionListItem term="Rationale" desc={rationale} />
                    <DescriptionListItem term="Guidance" desc={remediation} />
                </DescriptionList>
                {hasAnyNotifiers && (
                    <>
                        <Divider component="div" className="pf-v5-u-mt-md" />
                        <Title headingLevel="h3" className="pf-v5-u-pt-md pf-v5-u-pb-sm">
                            Notification routes
                        </Title>
                        <Grid hasGutter sm={12} md={6}>
                            {notifierIds?.map((notifierId) => {
                                const { name: notifierName, typeLabel } =
                                    getNotifierInfo(notifierId);
                                return (
                                    <GridItem key={notifierId}>
                                        <Card isFlat>
                                            <CardBody>
                                                <DescriptionList isCompact isHorizontal>
                                                    <DescriptionListItem
                                                        term="Notifier"
                                                        desc={notifierName}
                                                    />
                                                    {typeLabel && (
                                                        <DescriptionListItem
                                                            term="Type"
                                                            desc={typeLabel}
                                                        />
                                                    )}
                                                    <DescriptionListItem
                                                        term="Scope"
                                                        desc={
                                                            <Label color="blue">All scopes</Label>
                                                        }
                                                    />
                                                </DescriptionList>
                                            </CardBody>
                                        </Card>
                                    </GridItem>
                                );
                            })}
                            {notifierToCollectionMappings?.map((binding) => {
                                const { name: notifierName, typeLabel } = getNotifierInfo(
                                    binding.notifierId
                                );
                                return (
                                    <GridItem
                                        key={`${binding.notifierId}-${binding.collectionId}`}
                                    >
                                        <Card isFlat>
                                            <CardBody>
                                                <DescriptionList isCompact isHorizontal>
                                                    <DescriptionListItem
                                                        term="Notifier"
                                                        desc={notifierName}
                                                    />
                                                    {typeLabel && (
                                                        <DescriptionListItem
                                                            term="Type"
                                                            desc={typeLabel}
                                                        />
                                                    )}
                                                    <DescriptionListItem
                                                        term="Scope"
                                                        desc={
                                                            <Label color="green">
                                                                {binding.collectionName ||
                                                                    binding.collectionId}
                                                            </Label>
                                                        }
                                                    />
                                                </DescriptionList>
                                            </CardBody>
                                        </Card>
                                    </GridItem>
                                );
                            })}
                        </Grid>
                    </>
                )}
                <Divider component="div" className="pf-v5-u-mt-md" />
                <Title headingLevel="h3" className="pf-v5-u-mb-md pf-v5-u-pt-lg">
                    MITRE ATT&CK
                </Title>
                <MitreAttackVectorsViewContainer
                    policyId={policy.id}
                    policyFormMitreAttackVectors={policy.mitreAttackVectors}
                />
            </CardBody>
        </Card>
    );
}

export default PolicyOverview;
