import React from 'react';
import PropTypes from 'prop-types';
import uniqBy from 'lodash/uniqBy';

import Explanation from 'Containers/Violations/Enforcement/Explanation';
import Header from 'Containers/Violations/Enforcement/Header';
import LIFECYCLE_STAGES from 'constants/lifecycleStages';
import { ENFORCEMENT_ACTIONS } from 'constants/enforcementActions';

function getRuntimeEnforcementCount(processViolation) {
    return uniqBy(processViolation.processes, 'podId').length;
}

function EnforcementDetails({ alert }) {
    if (!alert?.enforcement) {
        return null;
    }

    const { lifecycleStage, processViolation, enforcement, policy } = alert;
    let enforcementCount = 0;
    if (lifecycleStage === LIFECYCLE_STAGES.RUNTIME) {
        if (enforcement?.action === ENFORCEMENT_ACTIONS.KILL_POD_ENFORCEMENT) {
            enforcementCount =
                enforcement && processViolation?.processes
                    ? getRuntimeEnforcementCount(processViolation)
                    : 0;
        } else {
            enforcementCount = 1;
        }
    } else if (lifecycleStage === LIFECYCLE_STAGES.DEPLOY) {
        enforcementCount = 1;
    }

    return (
        <div className="flex flex-col w-full overflow-auto pb-5">
            <div className="px-3 pt-5">
                <div className="bg-base-100 shadow">
                    <Header
                        lifecycleStage={alert.lifecycleStage}
                        enforcementCount={enforcementCount}
                        enforcementAction={enforcement?.action}
                    />
                    {enforcement && enforcementCount && (
                        <Explanation
                            lifecycleStage={lifecycleStage}
                            enforcement={enforcement}
                            policyId={policy.id}
                        />
                    )}
                </div>
            </div>
        </div>
    );
}

EnforcementDetails.propTypes = {
    alert: PropTypes.shape({
        lifecycleStage: PropTypes.string.isRequired,
        processViolation: PropTypes.shape({
            processes: PropTypes.shape({}),
        }),
        enforcement: PropTypes.shape({
            action: PropTypes.string,
        }),
        policy: PropTypes.shape({
            id: PropTypes.string.isRequired,
        }).isRequired,
    }).isRequired,
};

export default EnforcementDetails;
