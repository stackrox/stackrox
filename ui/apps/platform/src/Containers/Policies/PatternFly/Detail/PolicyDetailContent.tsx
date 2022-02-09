import React, { useState, useEffect } from 'react';
import { Formik } from 'formik';
import { Title, Divider } from '@patternfly/react-core';

import { fetchClustersAsArray } from 'services/ClustersService';
import { fetchNotifierIntegrations } from 'services/NotifierIntegrationsService';
import { Cluster } from 'types/cluster.proto';
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
    const [clusters, setClusters] = useState<Cluster[]>([]);
    const [notifiers, setNotifiers] = useState<NotifierIntegration[]>([]);

    useEffect(() => {
        fetchClustersAsArray()
            .then((data) => {
                setClusters(data as Cluster[]);
            })
            .catch(() => {
                // TODO
            });
    }, []);

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
            {(scope?.length > 0 || exclusions?.length > 0) && (
                <>
                    <Title headingLevel="h2" className="pf-u-mb-md pf-u-pt-lg">
                        Policy scope
                    </Title>
                    <Divider component="div" />
                    <PolicyScopeSection scope={scope} exclusions={exclusions} clusters={clusters} />
                </>
            )}
        </div>
    );
}

export default PolicyDetailContent;
