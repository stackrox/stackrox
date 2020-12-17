import React from 'react';
import { Message } from '@stackrox/ui-components';

function WarningMessage(policyDisabled) {
    let message = '';
    if (policyDisabled) {
        message =
            'This policy is not currently enabled. If enabled, the policy would generate violations for the following deployments on your system.';
    } else {
        message =
            'The policy settings you have selected will generate violations for the following deployments on your system. Please verify that this seems accurate before saving.';
    }
    return <Message type="warn">{message}</Message>;
}

export default WarningMessage;
