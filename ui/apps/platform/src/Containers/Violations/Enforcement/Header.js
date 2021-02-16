import React from 'react';
import PropTypes from 'prop-types';
import LIFECYCLE_STAGES from 'constants/lifecycleStages';
import { ENFORCEMENT_ACTIONS, ENFORCEMENT_ACTIONS_AS_STRING } from 'constants/enforcementActions';

function deployHeader(count) {
    let message = '';
    if (count && count > 0) {
        message = 'on this deployment is enabled for this policy';
    } else {
        message = 'on this deployment is not enabled for this policy';
    }
    return message;
}

function runtimeHeader(enforcementAction, enforcementCount) {
    if (!enforcementCount || enforcementAction === ENFORCEMENT_ACTIONS.UNSET_ENFORCEMENT) {
        return `on this deployment was not enabled as of the last known violation of this policy.`;
    }

    if (enforcementAction === ENFORCEMENT_ACTIONS.KILL_POD_ENFORCEMENT) {
        if (enforcementCount === 1) {
            return `"${ENFORCEMENT_ACTIONS_AS_STRING[enforcementAction]}" has been applied once`;
        }
        if (enforcementCount > 1) {
            return `"${ENFORCEMENT_ACTIONS_AS_STRING[enforcementAction]}" has been applied ${enforcementCount} times`;
        }
    }
    // For runtime violations other than process violations, the enforcement count is not tracked.
    return `${ENFORCEMENT_ACTIONS_AS_STRING[enforcementAction]} has been applied`;
}

function Header({ lifecycleStage, enforcementCount, enforcementAction }) {
    let countMessage = '';
    if (lifecycleStage === LIFECYCLE_STAGES.DEPLOY) {
        countMessage = deployHeader(enforcementCount);
    } else if (lifecycleStage === LIFECYCLE_STAGES.RUNTIME) {
        countMessage = runtimeHeader(enforcementAction, enforcementCount);
    }

    return (
        <div
            className="p-3 pb-2 border-b border-base-300 text-base-600 font-700 text-lg leading-normal"
            data-testid="enforcement-detail-message"
        >
            Enforcement {countMessage}
        </div>
    );
}

Header.propTypes = {
    lifecycleStage: PropTypes.string.isRequired,
    enforcementCount: PropTypes.number,
    enforcementAction: PropTypes.string,
};

Header.defaultProps = {
    enforcementCount: 0,
    enforcementAction: ENFORCEMENT_ACTIONS.UNSET_ENFORCEMENT,
};

export default Header;
