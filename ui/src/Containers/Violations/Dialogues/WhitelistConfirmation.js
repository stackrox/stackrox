import React from 'react';
import PropTypes from 'prop-types';
import { whitelistDeployments } from 'services/PoliciesService';
import pluralize from 'pluralize';
import Dialog from 'Components/Dialog';
import dialogues from '../dialogues';

// Filter the alerts displayed down to the ones checked, and group them into a map from policy ID to a list of
// deployment names, then whitelist every policy ID, deployment name pair in the map.
function whitelistAlerts(checkedAlertIds, alerts) {
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
        Object.keys(policyToDeployments).map(policyId => {
            return whitelistDeployments(policyId, policyToDeployments[policyId]);
        })
    );
}

function WhitelistConfirmation({
    dialogue,
    setDialogue,
    alerts,
    checkedAlertIds,
    setCheckedAlertIds
}) {
    if (dialogue !== dialogues.whitelist) {
        return null;
    }

    function closeAndClear() {
        setDialogue(null);
        setCheckedAlertIds([]);
    }

    function whitelistDeploymentsAction() {
        whitelistAlerts(checkedAlertIds, alerts).then(closeAndClear, closeAndClear);
    }

    function close() {
        setDialogue(null);
    }

    const numSelectedRows = checkedAlertIds.length;
    return (
        <Dialog
            isOpen={dialogue === dialogues.whitelist}
            text={`Are you sure you want to whitelist ${numSelectedRows} ${pluralize(
                'violation',
                numSelectedRows
            )}?`}
            onConfirm={whitelistDeploymentsAction}
            onCancel={close}
        />
    );
}

WhitelistConfirmation.propTypes = {
    dialogue: PropTypes.string,
    setDialogue: PropTypes.func.isRequired,
    alerts: PropTypes.arrayOf(
        PropTypes.shape({
            policy: PropTypes.shape({
                id: PropTypes.string.isRequired
            }).isRequired,
            deployment: PropTypes.shape({
                name: PropTypes.string.isRequired
            }).isRequired
        })
    ).isRequired,
    checkedAlertIds: PropTypes.arrayOf(PropTypes.string).isRequired,
    setCheckedAlertIds: PropTypes.func.isRequired
};

WhitelistConfirmation.defaultProps = {
    dialogue: undefined
};

export default WhitelistConfirmation;
