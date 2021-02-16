import React from 'react';
import PropTypes from 'prop-types';
import { Link } from 'react-router-dom';
import LIFECYCLE_STAGES from 'constants/lifecycleStages';

function getEnforcementExplanation(lifecycleStage, message) {
    if (lifecycleStage === LIFECYCLE_STAGES.DEPLOY) {
        return `Deployment data was evaluated against this StackRox policy. ${message}`;
    }

    if (lifecycleStage === LIFECYCLE_STAGES.RUNTIME) {
        return `Runtime data was evaluated against this StackRox policy. ${message}`;
    }

    return '';
}

function Explanation({ lifecycleStage, enforcement, policyId }) {
    const linkAddr = `../policies/${policyId}`;

    return (
        <div
            className="h-full p-3 text-base-600 font-600 text-base leading-loose"
            data-testid="enforcement-explanation-message"
        >
            <div className="pb-2">
                {getEnforcementExplanation(lifecycleStage, enforcement.message)}
            </div>
            <div className="pt-2 border-t">
                If the enforcement action is being applied several times, learn more on how you can
                <Link to={linkAddr}> remediate and resolve the issue.</Link>
            </div>
        </div>
    );
}

Explanation.propTypes = {
    policyId: PropTypes.string.isRequired,
    lifecycleStage: PropTypes.string.isRequired,
    enforcement: PropTypes.shape({
        message: PropTypes.string.isRequired,
    }).isRequired,
};

export default Explanation;
