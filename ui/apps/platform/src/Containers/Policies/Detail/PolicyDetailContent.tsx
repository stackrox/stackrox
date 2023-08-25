import React, { useState, useEffect } from 'react';
import { Formik } from 'formik';
import { Flex, Title, Divider, Grid } from '@patternfly/react-core';

import { fetchNotifierIntegrations } from 'services/NotifierIntegrationsService';
import { NotifierIntegration } from 'types/notifier.proto';
import { Policy } from 'types/policy.proto';
import PolicyOverview from './PolicyOverview';
import BooleanPolicyLogicSection from '../Wizard/Step3/BooleanPolicyLogicSection';
import PolicyScopeSection from './PolicyScopeSection';
import PolicyBehaviorSection from './PolicyBehaviorSection';

type PolicyDetailContentProps = {
    policy: Policy;
    isReview?: boolean;
};

function PolicyDetailContent({
    policy,
    isReview = false,
}: PolicyDetailContentProps): React.ReactElement {
    const [notifiers, setNotifiers] = useState<NotifierIntegration[]>([]);

    useEffect(() => {
        fetchNotifierIntegrations()
            .then((data) => {
                setNotifiers(data as NotifierIntegration[]);
            })
            .catch(() => {
                // TODO
            });
    }, []);

    const { enforcementActions, eventSource, exclusions, scope, lifecycleStages } = policy;
    return (
        <div data-testid="policy-details">
            <Flex direction={{ default: 'column' }}>
                <PolicyOverview policy={policy} notifiers={notifiers} isReview={isReview} />
                <Title headingLevel="h3" className="pf-u-mb-md pf-u-pt-lg">
                    Policy behavior
                </Title>
                <Divider component="div" className="pf-u-mb-md" />
                <PolicyBehaviorSection
                    lifecycleStages={lifecycleStages}
                    eventSource={eventSource}
                    enforcementActions={enforcementActions}
                />
                <Formik initialValues={policy} onSubmit={() => {}}>
                    {() => (
                        <>
                            <Title headingLevel="h3" className="pf-u-mb-md pf-u-pt-lg">
                                Policy criteria
                            </Title>
                            <Divider component="div" />
                            {/* this grid component specifies a GridItem to span 5 columns by default for policy sections */}
                            <Grid hasGutter lg={5}>
                                <BooleanPolicyLogicSection readOnly />
                            </Grid>
                        </>
                    )}
                </Formik>
                {(scope?.length > 0 || exclusions?.length > 0) && (
                    <>
                        <Title headingLevel="h3" className="pf-u-mb-md pf-u-pt-lg">
                            Policy scope
                        </Title>
                        <Divider component="div" />
                        <PolicyScopeSection scope={scope} exclusions={exclusions} />
                    </>
                )}
            </Flex>
        </div>
    );
}

export default PolicyDetailContent;
