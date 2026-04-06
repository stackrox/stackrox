import { useEffect, useState } from 'react';
import type { ReactElement } from 'react';
import { Bullseye, Spinner } from '@patternfly/react-core';

import { policiesBasePath } from 'routePaths';
import NotFoundMessage from 'Components/NotFoundMessage';
import PageTitle from 'Components/PageTitle';
import useURLSearch from 'hooks/useURLSearch';
import {
    generatePolicyFromSearch,
    getPolicy,
    updatePolicyDisabledState,
} from 'services/PoliciesService';
import type { ClientPolicy } from 'types/policy.proto';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import type { ExtendedPageAction } from 'utils/queryStringUtils';
import { getHasSearchApplied, getRequestQueryStringForSearchFilter } from 'utils/searchUtils';

import { getClientWizardPolicy, initialPolicy } from './policies.utils';
import PolicyDetail from './Detail/PolicyDetail';
import PolicyWizard from './Wizard/PolicyWizard';

import GeneratedPolicyErrorModal from './GeneratedPolicyErrorModal';
import type { ErrorsForGeneratedPolicy } from './GeneratedPolicyErrorModal';

function clonePolicy(policy: ClientPolicy): ClientPolicy {
    /*
     * Default policies will have the "criteriaLocked" and "mitreVectorsLocked" fields set to true.
     * When we clone these policies, we'll need to set them to false to allow users to edit
     * both the policy criteria and mitre attack vectors
     */
    return {
        ...policy,
        source: 'IMPERATIVE',
        criteriaLocked: false,
        id: '',
        isDefault: false,
        mitreVectorsLocked: false,
        name: `${policy.name} (COPY)`,
    };
}

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
    const { searchFilter } = useURLSearch();
    const [policy, setPolicy] = useState<ClientPolicy>(initialPolicy);
    const [errorsForGeneratedPolicy, setErrorsForGeneratedPolicy] =
        useState<ErrorsForGeneratedPolicy | null>(null);
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
        } else if (pageAction === 'generate' && getHasSearchApplied(searchFilter)) {
            generatePolicyFromSearch(getRequestQueryStringForSearchFilter(searchFilter))
                .then((data) => {
                    /*
                    // Note these omissions may not be handled gracefully by the policy form, but represent what the actual
                    // runtime value is at the time this file was converted to TypeScript.
                    policy: Omit<
                        ClientPolicy,
                        | 'severity'
                        | 'excludedImageNames'
                        | 'excludedDeploymentScopes'
                        | 'serverPolicySections'
                        | 'policySections'
                    > & { severity: PolicySeverity | null }
                    */

                    const alteredSearchTerms =
                        Array.isArray(data.alteredSearchTerms) &&
                        data.alteredSearchTerms.length !== 0
                            ? data.alteredSearchTerms
                            : [];
                    const hasNestedFields = Boolean(data.hasNestedFields);

                    setErrorsForGeneratedPolicy(
                        alteredSearchTerms.length !== 0 || hasNestedFields
                            ? {
                                  alteredSearchTerms,
                                  errorFromCatch: '',
                                  hasNestedFields,
                              }
                            : null
                    );

                    setPolicy(getClientWizardPolicy(data.policy ?? {}));
                })
                .catch((err) => {
                    // to get the actual error returned by the server, we have to dereference the response object first
                    //   because err.message is the generic Axios error message,
                    //   https://github.com/axios/axios/issues/960#issuecomment-309287911
                    setErrorsForGeneratedPolicy({
                        alteredSearchTerms: [],
                        errorFromCatch: err?.response?.data || 'An unrecognized error occurred.',
                        hasNestedFields: false,
                    });
                });
        }
    }, [pageAction, policyId, searchFilter]);

    function handleUpdateDisabledState(id: string, disabled: boolean) {
        return updatePolicyDisabledState(id, disabled).then(() => {
            /*
             * If success, render PolicyDetail element with updated policy.
             * If failure, PolicyDetail element has catch block to display error.
             */
            return getPolicy(id).then((data) => {
                setPolicy(getClientWizardPolicy(data));
            });
        });
    }

    return (
        <>
            <PageTitle title="Policy Management - Policy" />
            {isLoading ? (
                <Bullseye>
                    <Spinner />
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
            {errorsForGeneratedPolicy !== null && (
                <GeneratedPolicyErrorModal
                    errors={errorsForGeneratedPolicy}
                    onClose={() => setErrorsForGeneratedPolicy(null)}
                />
            )}
        </>
    );
}

export default PolicyPage;
