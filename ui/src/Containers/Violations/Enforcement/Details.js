import React from 'react';
import PropTypes from 'prop-types';
import uniqBy from 'lodash/uniqBy';

import Explanation from 'Containers/Violations/Enforcement/Explanation';
import Header from 'Containers/Violations/Enforcement/Header';

function getRuntimeEnforcementCount(processViolation) {
    return uniqBy(processViolation.processes, 'podId').length;
}

function EnforcementDetails({ alert }) {
    if (!alert) return null;
    const { lifecycleStage, processViolation, enforcement, policy } = alert;
    let enforcementCount = 0;
    if (lifecycleStage === 'RUNTIME') {
        enforcementCount = enforcement ? getRuntimeEnforcementCount(processViolation) : 0;
    } else if (lifecycleStage === 'DEPLOY') {
        enforcementCount = !!enforcement;
    }
    return (
        <div className="flex flex-col w-full overflow-auto pb-5">
            <div className="px-3 pt-5">
                <div className="bg-base-100 shadow">
                    <Header
                        lifecycleStage={alert.lifecycleStage}
                        enforcementCount={enforcementCount}
                    />
                    {enforcement && enforcementCount && (
                        <Explanation lifecycleStage={lifecycleStage} policyId={policy.id} />
                    )}
                </div>
            </div>
        </div>
    );
}

EnforcementDetails.propTypes = {
    alert: PropTypes.shape({
        lifecycleStage: PropTypes.string.isRequired,
        processViolation: PropTypes.shape({}),
        enforcement: PropTypes.shape({}),
        policy: PropTypes.shape({
            id: PropTypes.string.isRequired
        }).isRequired
    }).isRequired
};

export default EnforcementDetails;
