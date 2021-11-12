import React, { ReactElement, useEffect, useState } from 'react';
import { Alert, Bullseye, PageSection, Spinner } from '@patternfly/react-core';

import { getPolicy } from 'services/PoliciesService';
import { Policy } from 'types/policy.proto';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import { PageAction } from './policies.utils';
import PolicyDetail from './Detail/PolicyDetail';
import PolicyWizard from './Wizard/PolicyWizard';

const initialPolicy: Policy = {
    id: '',
    name: '',
    description: '',
    severity: 'LOW_SEVERITY',
    disabled: false,
    lifecycleStages: [],
    notifiers: [],
    lastUpdated: null,
    eventSource: 'NOT_APPLICABLE',
    isDefault: false,
    rationale: '',
    remediation: '',
    categories: [],
    fields: null,
    exclusions: [],
    scope: [],
    enforcementActions: [],
    SORT_name: '', // For internal use only.
    SORT_lifecycleStage: '', // For internal use only.
    SORT_enforcement: false, // For internal use only.
    policyVersion: '',
    policySections: [],
    mitreAttackVectors: [],
    criteriaLocked: false,
    mitreVectorsLocked: false,
};

type PolicyPageProps = {
    pageAction?: PageAction;
    policyId?: string;
};

function PolicyPage({ pageAction, policyId }: PolicyPageProps): ReactElement {
    const [policy, setPolicy] = useState<Policy>(initialPolicy);
    const [policyError, setPolicyError] = useState<ReactElement | null>(null);
    const [isLoading, setIsLoading] = useState(false);

    useEffect(() => {
        setPolicyError(null);
        if (policyId) {
            // action is 'edit' or undefined
            setIsLoading(true);
            getPolicy(policyId)
                .then((data) => {
                    setPolicy(data);
                })
                .catch((error) => {
                    setPolicy(initialPolicy);
                    setPolicyError(
                        <Alert title="Request failure for policy" variant="danger" isInline>
                            {getAxiosErrorMessage(error)}
                        </Alert>
                    );
                })
                .finally(() => {
                    setIsLoading(false);
                });
        } else {
            // action is 'create'
            setPolicy(initialPolicy);
        }
    }, [policyId]);

    return (
        <PageSection variant="light" isFilled id="policy-page">
            {isLoading ? (
                <Bullseye>
                    <Spinner />
                </Bullseye>
            ) : (
                policyError || // TODO ROX-8487: Improve PolicyPage when request fails
                (pageAction ? (
                    <PolicyWizard pageAction={pageAction} policy={policy} />
                ) : (
                    <PolicyDetail policy={policy} />
                ))
            )}
        </PageSection>
    );
}

export default PolicyPage;
