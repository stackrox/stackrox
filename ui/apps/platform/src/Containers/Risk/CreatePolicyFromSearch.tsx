import { useContext, useEffect } from 'react';
import { useNavigate } from 'react-router-dom-v5-compat';
import { connect } from 'react-redux';
import { Button } from '@patternfly/react-core';

import workflowStateContext from 'Containers/workflowStateContext';
import { actions as formMessageActions } from 'reducers/formMessages';
import { actions as notificationActions } from 'reducers/notifications';
import { actions as wizardActions } from 'reducers/policies/wizard';
import { generatePolicyFromSearch } from 'services/PoliciesService';
import searchOptionsToQuery from 'services/searchOptionsToQuery';
import { convertToRestSearch } from 'utils/searchUtils';
import { policiesBasePath } from 'routePaths';
import type { ClientPolicy, PolicySeverity } from 'types/policy.proto';

type CreatePolicyFromSearchProps = {
    setWizardPolicy: (
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
    ) => void;
    addToast: (message: string) => void;
    removeToast: () => void;
    addFormMessage: (message: { type: string; message: string }) => void;
    clearFormMessages: () => void;
};

function CreatePolicyFromSearch({
    setWizardPolicy,
    addToast,
    removeToast,
    addFormMessage,
    clearFormMessages,
}: CreatePolicyFromSearchProps) {
    const workflowState = useContext(workflowStateContext);
    const navigate = useNavigate();

    // this utility filters out incomplete search pairs
    const currentSearch = workflowState.getCurrentSearchState();
    const policySearchOptions = convertToRestSearch(currentSearch);
    // ensure clean slate for policy form messages
    useEffect(() => {
        clearFormMessages();
    }, [clearFormMessages]);

    function createPolicyFromSearch() {
        const queryString = searchOptionsToQuery(policySearchOptions);

        generatePolicyFromSearch(queryString)
            .then((response) => {
                navigate(`${policiesBasePath}?action=generate`);

                const newPolicy = {
                    ...response?.policy,
                    severity: null,
                };

                if (response.alteredSearchTerms?.length) {
                    const termsRemoved = response.alteredSearchTerms.join(', ');
                    const message = `The following search terms were removed or altered when creating the policy: ${termsRemoved}`;

                    addFormMessage({ type: 'warn', message });
                }

                if (response?.hasNestedFields) {
                    addFormMessage({
                        type: 'warn',
                        message: 'Policy contained nested fields.',
                    });
                }

                setWizardPolicy(newPolicy);
            })
            .catch((err) => {
                // to get the actual error returned by the server, we have to dereference the response object first
                //   because err.message is the generic Axios error message,
                //   https://github.com/axios/axios/issues/960#issuecomment-309287911
                const serverErr = err?.response?.data || 'An unrecognized error occurred.';

                addToast(`Could not create a policy from this search: ${serverErr.message}`);
                setTimeout(removeToast, 10000);
            });
    }

    const isPolicyBtnDisabled = !policySearchOptions?.length;

    return (
        <Button
            variant="secondary"
            className="ml-4"
            onClick={createPolicyFromSearch}
            isDisabled={isPolicyBtnDisabled}
        >
            Create policy
        </Button>
    );
}

const mapDispatchToProps = {
    setWizardPolicy: wizardActions.setWizardPolicy,
    addToast: notificationActions.addNotification,
    removeToast: notificationActions.removeOldestNotification,
    addFormMessage: formMessageActions.addFormMessage,
    clearFormMessages: formMessageActions.clearFormMessages,
};

export default connect(null, mapDispatchToProps)(CreatePolicyFromSearch);
