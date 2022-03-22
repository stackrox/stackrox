import React, { ReactElement, useEffect, useState } from 'react';
import { useSelector } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { Bullseye, Spinner } from '@patternfly/react-core';

import { selectors } from 'reducers';
import { policiesBasePath } from 'routePaths';
import NotFoundMessage from 'Components/NotFoundMessage';
import PageTitle from 'Components/PageTitle';
import { getPolicy, updatePolicyDisabledState } from 'services/PoliciesService';
import { Policy } from 'types/policy.proto';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import { ExtendedPageAction } from 'utils/queryStringUtils';

import { getClientWizardPolicy } from './policies.utils';
import PolicyDetail from './Detail/PolicyDetail';
import PolicyWizard from './Wizard/PolicyWizard';

function clonePolicy(policy: Policy) {
    /*
     * Default policies will have the "criteriaLocked" and "mitreVectorsLocked" fields set to true.
     * When we clone these policies, we'll need to set them to false to allow users to edit
     * both the policy criteria and mitre attack vectors
     */
    return {
        ...policy,
        criteriaLocked: false,
        id: '',
        isDefault: false,
        mitreVectorsLocked: false,
        name: `${policy.name} (COPY)`,
    };
}

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
    excludedImageNames: [],
    excludedDeploymentScopes: [],
    SORT_name: '', // For internal use only.
    SORT_lifecycleStage: '', // For internal use only.
    SORT_enforcement: false, // For internal use only.
    policyVersion: '',
    serverPolicySections: [],
    policySections: [
        {
            sectionName: 'Policy Section 1',
            policyGroups: [],
        },
    ],
    mitreAttackVectors: [],
    criteriaLocked: false,
    mitreVectorsLocked: false,
};

type WizardPolicyState = {
    wizardPolicy: Policy;
};

const wizardPolicyState = createStructuredSelector<WizardPolicyState, { wizardPolicy: Policy }>({
    wizardPolicy: selectors.getWizardPolicy,
});

type PolicyPageProps = {
    hasWriteAccessForPolicy: boolean;
    pageAction?: ExtendedPageAction;
    policyId?: string;
};

function PolicyPage({
    hasWriteAccessForPolicy,
    pageAction,
    policyId,
}: PolicyPageProps): ReactElement {
    const { wizardPolicy } = useSelector(wizardPolicyState);

    const [policy, setPolicy] = useState<Policy>(
        pageAction === 'generate' && wizardPolicy
            ? getClientWizardPolicy(wizardPolicy)
            : initialPolicy
    );
    const [policyError, setPolicyError] = useState<ReactElement | null>(null);
    const [isLoading, setIsLoading] = useState(false);

    useEffect(() => {
        setPolicyError(null);
        if (policyId) {
            // action is 'clone' or 'edit' or undefined
            setIsLoading(true);
            getPolicy(policyId)
                .then((data) => {
                    const clientWizardPolicy = getClientWizardPolicy(data);
                    setPolicy(
                        pageAction === 'clone'
                            ? clonePolicy(clientWizardPolicy)
                            : clientWizardPolicy
                    );
                })
                .catch((error) => {
                    setPolicy(initialPolicy);
                    setPolicyError(
                        <NotFoundMessage
                            title="404: We couldn't find that page"
                            message={getAxiosErrorMessage(error)}
                            actionText="Go to Policies"
                            url={policiesBasePath}
                        />
                    );
                })
                .finally(() => {
                    setIsLoading(false);
                });
        }
    }, [pageAction, policyId]);

    function handleUpdateDisabledState(id: string, disabled: boolean) {
        return updatePolicyDisabledState(id, disabled).then(() => {
            /*
             * If success, render PolicyDetail element with updated policy.
             * If failure, PolicyDetail element has catch block to display error.
             */
            return getPolicy(id).then((data) => {
                setPolicy(data);
            });
        });
    }

    return (
        <>
            <PageTitle title="Policies - Policy" />
            {isLoading ? (
                <Bullseye>
                    <Spinner isSVG />
                </Bullseye>
            ) : (
                policyError || // TODO ROX-8487: Improve PolicyPage when request fails
                (pageAction ? (
                    <PolicyWizard pageAction={pageAction} policy={policy} />
                ) : (
                    <PolicyDetail
                        handleUpdateDisabledState={handleUpdateDisabledState}
                        hasWriteAccessForPolicy={hasWriteAccessForPolicy}
                        policy={policy}
                    />
                ))
            )}
        </>
    );
}

export default PolicyPage;
