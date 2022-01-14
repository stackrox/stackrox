import React from 'react';
import { Formik } from 'formik';
import { Title, Divider } from '@patternfly/react-core';

import { Cluster } from 'types/cluster.proto';
import { NotifierIntegration } from 'types/notifier.proto';
import { Policy } from 'types/policy.proto';
import MitreAttackVectorsView from 'Containers/MitreAttackVectors/MitreAttackVectorsView';
import PolicyOverview from './PolicyOverview';
import BooleanPolicyLogicSection from '../Wizard/Step3/BooleanPolicyLogicSection';

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
    return (
        <Formik initialValues={policy} onSubmit={() => {}}>
            {() => (
                <>
                    <PolicyOverview clusters={clusters} policy={policy} notifiers={notifiers} />
                    <Title headingLevel="h2" className="pf-u-mb-md pf-u-pt-lg">
                        MITRE ATT&CK
                    </Title>
                    <Divider component="div" />
                    <MitreAttackVectorsView policyId={policy.id} />
                    <Title headingLevel="h2" className="pf-u-mb-md">
                        Policy criteria
                    </Title>
                    <Divider component="div" className="pf-u-pb-md" />
                    <BooleanPolicyLogicSection readOnly />
                </>
            )}
        </Formik>
    );
}

export default PolicyDetailContent;
