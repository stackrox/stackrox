import React from 'react';

import Message from 'Components/Message';

function WarningMessage(policyDisabled) {
    let message = '';
    if (policyDisabled) {
        message =
            'This policy is not currently enabled. If enabled, the policy would generate violations for the following deployments on your system.';
    } else {
        message =
            'The policy settings you have selected will generate violations for the following deployments on your system, Please verify that this seems accurate before saving.';
    }
    return <Message message={message} type="warn" />;
}

export default WarningMessage;
