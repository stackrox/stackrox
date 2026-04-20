import { useEffect, useState } from 'react';
import type { ReactElement } from 'react';
import { Formik } from 'formik';
import { Flex, Grid, Stack, Title } from '@patternfly/react-core';

import { fetchNotifierIntegrations } from 'services/NotifierIntegrationsService';
import type { NotifierIntegration } from 'types/notifier.proto';
import type { ClientPolicy } from 'types/policy.proto';
import PolicyOverview from './PolicyOverview';
import BooleanPolicyLogicSection from '../Wizard/Step3/BooleanPolicyLogicSection';
import PolicyScopeSection from './PolicyScopeSection';
import PolicyBehaviorSection from './PolicyBehaviorSection';
import { getExcludedDeployments, getExcludedImageNames } from '../policies.utils';

type PolicyDetailContentProps = {
    policy: ClientPolicy;
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

    const {
        enforcementActions,
        eventSource,
        exclusions,
        scope,
        lifecycleStages,
        excludedDeploymentScopes = [],
        excludedImageNames = [],
    } = policy;
    const hasWizardDeploymentScopes = excludedDeploymentScopes.some((d) => d.name || d.scope);
    const hasWizardImageNames = excludedImageNames.some((name) => name !== '');
    const showPolicyScopeSection =
        (scope?.length ?? 0) > 0 ||
        getExcludedDeployments(exclusions).length > 0 ||
        getExcludedImageNames(exclusions).length > 0 ||
        hasWizardDeploymentScopes ||
        hasWizardImageNames;

    return (
        <div data-testid="policy-details">
            <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsLg' }}>
                <Stack hasGutter>
                    <Title headingLevel="h2">Policy overview</Title>
                    <PolicyOverview policy={policy} notifiers={notifiers} isReview={isReview} />
                </Stack>
                <Stack hasGutter>
                    <Title headingLevel="h2">Policy behavior</Title>
                    <PolicyBehaviorSection
                        lifecycleStages={lifecycleStages}
                        eventSource={eventSource}
                        enforcementActions={enforcementActions}
                    />
                </Stack>
                <Stack hasGutter>
                    <Formik initialValues={policy} onSubmit={() => {}}>
                        {() => (
                            <>
                                <Title headingLevel="h2">Policy criteria</Title>
                                {/* this grid component specifies a GridItem to span 5 columns by default for policy sections */}
                                <Grid hasGutter lg={5}>
                                    <BooleanPolicyLogicSection readOnly />
                                </Grid>
                            </>
                        )}
                    </Formik>
                </Stack>
                {showPolicyScopeSection && (
                    <Stack hasGutter>
                        <Title headingLevel="h2">Policy scope</Title>
                        <PolicyScopeSection
                            scope={scope}
                            exclusions={exclusions}
                            excludedDeploymentScopes={excludedDeploymentScopes}
                            excludedImageNames={excludedImageNames}
                        />
                    </Stack>
                )}
            </Flex>
        </div>
    );
}

export default PolicyDetailContent;
