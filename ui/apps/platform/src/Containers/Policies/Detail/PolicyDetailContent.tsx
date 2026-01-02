import { useEffect, useState } from 'react';
import type { ReactElement } from 'react';
import { Formik } from 'formik';
import { Flex, Grid, Title } from '@patternfly/react-core';

import { fetchNotifierIntegrations } from 'services/NotifierIntegrationsService';
import type { NotifierIntegration } from 'types/notifier.proto';
import type { BasePolicy } from 'types/policy.proto';
import PolicyOverview from './PolicyOverview';
import BooleanPolicyLogicSection from '../Wizard/Step3/BooleanPolicyLogicSection';
import PolicyScopeSection from './PolicyScopeSection';
import PolicyBehaviorSection from './PolicyBehaviorSection';

type PolicyDetailContentProps = {
    policy: BasePolicy;
    isReview?: boolean;
};

function PolicyDetailContent({ policy, isReview = false }: PolicyDetailContentProps): ReactElement {
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
                <Title headingLevel="h2" className="pf-v6-u-mb-md">
                    Policy overview
                </Title>
                <PolicyOverview policy={policy} notifiers={notifiers} isReview={isReview} />
                <Title headingLevel="h2" className="pf-v6-u-mt-xl pf-v6-u-mb-md">
                    Policy behavior
                </Title>
                <PolicyBehaviorSection
                    lifecycleStages={lifecycleStages}
                    eventSource={eventSource}
                    enforcementActions={enforcementActions}
                />
                <Formik initialValues={policy} onSubmit={() => {}}>
                    {() => (
                        <>
                            <Title headingLevel="h2" className="pf-v6-u-mt-xl pf-v6-u-mb-md">
                                Policy criteria
                            </Title>
                            {/* this grid component specifies a GridItem to span 5 columns by default for policy sections */}
                            <Grid hasGutter lg={5}>
                                <BooleanPolicyLogicSection readOnly />
                            </Grid>
                        </>
                    )}
                </Formik>
                {(scope?.length > 0 || exclusions?.length > 0) && (
                    <>
                        <Title headingLevel="h2" className="pf-v6-u-mt-xl pf-v6-u-mb-md">
                            Policy scope
                        </Title>
                        <PolicyScopeSection scope={scope} exclusions={exclusions} />
                    </>
                )}
            </Flex>
        </div>
    );
}

export default PolicyDetailContent;
