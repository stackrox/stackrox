import React from 'react';
import { Formik } from 'formik';
import { Title, Divider } from '@patternfly/react-core';

import { Cluster } from 'types/cluster.proto';
import { NotifierIntegration } from 'types/notifier.proto';
import { Policy } from 'types/policy.proto';
import PolicyOverview from './PolicyOverview';
import BooleanPolicyLogicSection from '../Wizard/Step3/BooleanPolicyLogicSection';
import PolicyScopeSection from './PolicyScopeSection';
import PolicyBehaviorSection from './PolicyBehaviorSection';

type PolicyDetailContentProps = {
    clusters: Cluster[];
    policy: Policy;
    notifiers: NotifierIntegration[];
    isReview?: boolean;
};

function PolicyDetailContent({
    clusters,
    policy,
    notifiers,
    isReview = false,
}: PolicyDetailContentProps): React.ReactElement {
    const { enforcementActions, eventSource, exclusions, scope, lifecycleStages } = policy;
    return (
        <>
            <PolicyOverview policy={policy} notifiers={notifiers} isReview={isReview} />
            <Title headingLevel="h2" className="pf-u-mb-md pf-u-pt-lg">
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
                        <Title headingLevel="h2" className="pf-u-mb-md pf-u-pt-lg">
                            Policy criteria
                        </Title>
                        <Divider component="div" className="pf-u-pb-md" />
                        <BooleanPolicyLogicSection readOnly />
                    </>
                )}
            </Formik>
            {(scope.length > 0 || exclusions.length > 0) && (
                <>
                    <Title headingLevel="h2" className="pf-u-mb-md pf-u-pt-lg">
                        Policy scope
                    </Title>
                    <Divider component="div" />
                    <PolicyScopeSection scope={scope} exclusions={exclusions} clusters={clusters} />
                </>
            )}
        </>
    );
}

export default PolicyDetailContent;
