import React from 'react';
import PropTypes from 'prop-types';
import { resolveAlerts } from 'services/AlertsService';
import pluralize from 'pluralize';
import Dialog from 'Components/Dialog';
import dialogues from '../dialogues';

function ResolveConfirmation({
    dialogue,
    setDialogue,
    checkedAlertIds,
    setCheckedAlertIds,
    runtimeAlerts
}) {
    if (dialogue !== dialogues.resolve) {
        return null;
    }

    function closeAndClear() {
        setDialogue(null);
        setCheckedAlertIds([]);
    }

    function resolveAlertsAction() {
        const resolveSelection = checkedAlertIds.filter(id => runtimeAlerts.has(id));
        resolveAlerts(resolveSelection).then(closeAndClear, closeAndClear);
    }

    function close() {
        setDialogue(null);
    }

    const numSelectedRows = checkedAlertIds.reduce(
        (acc, id) => (runtimeAlerts.has(id) ? acc + 1 : acc),
        0
    );
    return (
        <Dialog
            isOpen={dialogue === dialogues.resolve}
            text={`Are you sure you want to resolve ${numSelectedRows} ${pluralize(
                'violation',
                numSelectedRows
            )}?`}
            onConfirm={resolveAlertsAction}
            onCancel={close}
        />
    );
}

ResolveConfirmation.propTypes = {
    dialogue: PropTypes.string,
    setDialogue: PropTypes.func.isRequired,
    checkedAlertIds: PropTypes.arrayOf(PropTypes.string).isRequired,
    setCheckedAlertIds: PropTypes.func.isRequired,
    runtimeAlerts: PropTypes.shape({
        has: PropTypes.func.isRequired
    }).isRequired
};

ResolveConfirmation.defaultProps = {
    dialogue: undefined
};

export default ResolveConfirmation;
