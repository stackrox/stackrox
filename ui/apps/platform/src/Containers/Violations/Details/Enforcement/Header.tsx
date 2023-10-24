import React, { ReactElement } from 'react';

import LIFECYCLE_STAGES from 'constants/lifecycleStages';
import { ENFORCEMENT_ACTIONS, ENFORCEMENT_ACTIONS_AS_STRING } from 'constants/enforcementActions';
import { LifecycleStage } from 'types/policy.proto';

function getDeployHeader(count) {
    let message = '';
    if (count && count > 0) {
        message = 'on this deployment is enabled for this policy';
    } else {
        message = 'on this deployment is not enabled for this policy';
    }
    return message;
}

function getRuntimeHeader(enforcementAction, enforcementCount) {
    if (!enforcementCount || enforcementAction === ENFORCEMENT_ACTIONS.UNSET_ENFORCEMENT) {
        return `on this deployment was not enabled as of the last known violation of this policy.`;
    }

    if (enforcementAction === ENFORCEMENT_ACTIONS.KILL_POD_ENFORCEMENT) {
        if (enforcementCount === 1) {
            return `"${
                ENFORCEMENT_ACTIONS_AS_STRING[enforcementAction] as string
            }" has been applied once`;
        }
        if (enforcementCount > 1) {
            return `"${
                ENFORCEMENT_ACTIONS_AS_STRING[enforcementAction] as string
            }" has been applied ${enforcementCount as string} times`;
        }
    }
    // For runtime violations other than process violations, the enforcement count is not tracked.
    return `${ENFORCEMENT_ACTIONS_AS_STRING[enforcementAction] as string} has been applied`;
}

type HeaderProps = {
    lifecycleStage: LifecycleStage;
    enforcementCount: number;
    enforcementAction: string;
};

function Header({
    lifecycleStage,
    enforcementCount,
    enforcementAction,
}: HeaderProps): ReactElement {
    let countMessage = '';
    if (lifecycleStage === LIFECYCLE_STAGES.DEPLOY) {
        countMessage = getDeployHeader(enforcementCount);
    } else if (lifecycleStage === LIFECYCLE_STAGES.RUNTIME) {
        countMessage = getRuntimeHeader(enforcementAction, enforcementCount);
    }

    return (
        <div className="pf-u-p-md" aria-label="Enforcement detail message">
            Enforcement {countMessage}
        </div>
    );
}

export default Header;
