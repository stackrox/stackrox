import React, { ReactElement } from 'react';
import {
    Card,
    CardBody,
    DescriptionList,
    Grid,
    GridItem,
    Title,
    Divider,
    CardHeader,
} from '@patternfly/react-core';

import DescriptionListItem from 'Components/DescriptionListItem';
import PolicySeverityIconText from 'Components/PatternFly/IconText/PolicySeverityIconText';
import { NotifierIntegration } from 'types/notifier.proto';
import { Policy } from 'types/policy.proto';
import MitreAttackVectorsViewContainer from 'Containers/MitreAttackVectors/MitreAttackVectorsViewContainer';

import { formatCategories, formatType } from '../policies.utils';
import Notifier from './Notifier';

type PolicyOverviewProps = {
    notifiers: NotifierIntegration[];
    policy: Policy;
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
        isDefault,
        notifiers: notifierIds,
        rationale,
        remediation,
        severity,
        name,
    } = policy;
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
                    <DescriptionListItem term="Type" desc={formatType(isDefault)} />
                    <DescriptionListItem term="Description" desc={description} />
                    <DescriptionListItem term="Rationale" desc={rationale} />
                    <DescriptionListItem term="Guidance" desc={remediation} />
                </DescriptionList>
                {notifierIds?.length !== 0 && (
                    <>
                        <Divider component="div" className="pf-u-mt-md" />
                        <Title headingLevel="h3" className="pf-u-pt-md pf-u-pb-sm">
                            Notifiers
                        </Title>
                        <Grid hasGutter sm={12} md={6}>
                            {notifierIds?.map((notifierId) => (
                                <GridItem key={notifierId}>
                                    <Card isFlat>
                                        <CardBody>
                                            <Notifier
                                                notifierId={notifierId}
                                                notifiers={notifiers}
                                            />
                                        </CardBody>
                                    </Card>
                                </GridItem>
                            ))}
                        </Grid>
                    </>
                )}
                <Divider component="div" className="pf-u-mt-md" />
                <Title headingLevel="h3" className="pf-u-mb-md pf-u-pt-lg">
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
