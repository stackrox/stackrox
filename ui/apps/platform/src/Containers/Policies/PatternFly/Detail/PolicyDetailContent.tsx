import React from 'react';
import { Formik } from 'formik';
import { Title, Divider } from '@patternfly/react-core';

import { Cluster } from 'types/cluster.proto';
import { NotifierIntegration } from 'types/notifier.proto';
import { Policy } from 'types/policy.proto';
import MitreAttackVectorsView from 'Containers/MitreAttackVectors/MitreAttackVectorsView';
import PolicyOverview from './PolicyOverview';
import BooleanPolicyLogicSection from '../Wizard/Step3/BooleanPolicyLogicSection';
import PolicyScopeSection from './PolicyScopeSection';
import { getExcludedDeployments, getExcludedImageNames } from '../policies.utils';

type PolicyDetailContentProps = {
    clusters: Cluster[];
    policy: Policy;
    notifiers: NotifierIntegration[];
};

function PolicyDetailContent({
    clusters,
    policy,
    notifiers,
}: PolicyDetailContentProps): React.ReactElement {
    const {
        categories,
        description,
        enforcementActions,
        eventSource,
        exclusions,
        scope,
        isDefault,
        lifecycleStages,
        notifiers: notifierIds,
        rationale,
        remediation,
        severity,
    } = policy;
    // const enforcementLifecycleStages = getEnforcementLifecycleStages(
    //     lifecycleStages,
    //     enforcementActions
    // );
    const excludedDeployments = getExcludedDeployments(exclusions);
    const excludedImageNames = getExcludedImageNames(exclusions);
    return (
        <>
            <PolicyOverview policy={policy} notifiers={notifiers} />
            <Title headingLevel="h2" className="pf-u-mb-md pf-u-pt-lg">
                Policy behavior
            </Title>
            <Divider component="div" />
            <div> behavior details here </div>
            {/* <Title headingLevel="h2" className="pf-u-mb-md pf-u-pt-lg">
                        MITRE ATT&CK
                    </Title>
                    <Divider component="div" />
                    <MitreAttackVectorsView policyId={policy.id} /> */}
            <Formik initialValues={policy} onSubmit={() => {}}>
                {() => (
                    <>
                        <Title headingLevel="h2" className="pf-u-mb-md">
                            Policy criteria
                        </Title>
                        <Divider component="div" className="pf-u-pb-md" />
                        <BooleanPolicyLogicSection readOnly />
                    </>
                )}
            </Formik>
            <Title headingLevel="h2" className="pf-u-mb-md pf-u-pt-lg">
                Policy scope
            </Title>
            <Divider component="div" />
            <PolicyScopeSection
                scope={scope}
                excludedDeployments={excludedDeployments}
                excludedImageNames={excludedImageNames}
                clusters={clusters}
            />
        </>
    );
}

export default PolicyDetailContent;
