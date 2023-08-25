import React, { useContext, useEffect } from 'react';
import PropTypes from 'prop-types';
import { useHistory } from 'react-router-dom';
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

function CreatePolicyFromSearch({
    setWizardPolicy,
    addToast,
    removeToast,
    addFormMessage,
    clearFormMessages,
}) {
    const workflowState = useContext(workflowStateContext);
    const history = useHistory();

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
                history.push({
                    pathname: policiesBasePath,
                    search: '?action=generate',
                });

                const newPolicy = {
                    ...response?.policy,
                    severity: null,
                };

                if (response.alteredSearchTerms?.length) {
                    const termsRemoved = response.alteredSearchTerms.join(', ');
                    const message = `The following search terms were removed or altered when creating the policy: ${termsRemoved}`;

                    addFormMessage({ type: 'warn', message });
                }

                if (response?.hasNestedField) {
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

CreatePolicyFromSearch.propTypes = {
    setWizardPolicy: PropTypes.func.isRequired,
    addToast: PropTypes.func.isRequired,
    removeToast: PropTypes.func.isRequired,
    addFormMessage: PropTypes.func.isRequired,
    clearFormMessages: PropTypes.func.isRequired,
};

const mapDispatchToProps = {
    setWizardPolicy: wizardActions.setWizardPolicy,
    addToast: notificationActions.addNotification,
    removeToast: notificationActions.removeOldestNotification,
    addFormMessage: formMessageActions.addFormMessage,
    clearFormMessages: formMessageActions.clearFormMessages,
};

export default connect(null, mapDispatchToProps)(CreatePolicyFromSearch);
