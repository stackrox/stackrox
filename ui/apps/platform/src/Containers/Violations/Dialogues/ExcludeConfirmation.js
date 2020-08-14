import React from 'react';
import PropTypes from 'prop-types';
import { excludeDeployments } from 'services/PoliciesService';
import pluralize from 'pluralize';
import Dialog from 'Components/Dialog';

// Filter the alerts displayed down to the ones checked, and group them into a map from policy ID to a list of
// deployment names, then exclude every policy ID, deployment name pair in the map.
function excludeAlerts(checkedAlertIds, alerts) {
    const checkedAlertsSet = new Set(checkedAlertIds);

    const policyToDeployments = {};
    alerts
        .filter(({ id }) => checkedAlertsSet.has(id))
        .forEach(({ policy, deployment }) => {
            if (!policyToDeployments[policy.id]) {
                policyToDeployments[policy.id] = [deployment.name];
            } else {
                policyToDeployments[policy.id].push(deployment.name);
            }
        });

    return Promise.all(
        Object.keys(policyToDeployments).map((policyId) => {
            return excludeDeployments(policyId, policyToDeployments[policyId]);
        })
    );
}

function ExcludeConfirmation({ setDialogue, alerts, checkedAlertIds, setCheckedAlertIds }) {
    function closeAndClear() {
        setDialogue(null);
        setCheckedAlertIds([]);
    }

    function excludeDeploymentsAction() {
        excludeAlerts(checkedAlertIds, alerts).then(closeAndClear, closeAndClear);
    }

    function close() {
        setDialogue(null);
    }

    const numSelectedRows = checkedAlertIds.length;
    return (
        <Dialog
            isOpen
            text={`Are you sure you want to exclude ${numSelectedRows} ${pluralize(
                'violation',
                numSelectedRows
            )}?`}
            onConfirm={excludeDeploymentsAction}
            onCancel={close}
        />
    );
}

ExcludeConfirmation.propTypes = {
    setDialogue: PropTypes.func.isRequired,
    alerts: PropTypes.arrayOf(
        PropTypes.shape({
            policy: PropTypes.shape({
                id: PropTypes.string.isRequired,
            }).isRequired,
            deployment: PropTypes.shape({
                name: PropTypes.string.isRequired,
            }).isRequired,
        })
    ).isRequired,
    checkedAlertIds: PropTypes.arrayOf(PropTypes.string).isRequired,
    setCheckedAlertIds: PropTypes.func.isRequired,
};

export default ExcludeConfirmation;
