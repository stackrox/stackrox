import React from 'react';
import PropTypes from 'prop-types';
import { Link } from 'react-router-dom';

import lifecycleToExplanation from 'Containers/Violations/Enforcement/descriptors';

function Explanation({ lifecycleStage, policyId }) {
    const linkAddr = `../policies/${policyId}`;
    return (
        <div
            className="h-full p-3 text-base-600 font-600 text-base leading-loose"
            data-test-id="enforcement-explanation-message"
        >
            <div className="pb-2">{lifecycleToExplanation[lifecycleStage]}</div>
            <div className="pt-2 border-t">
                If the enforcement action is being applied several times, learn more on how you can
                <Link to={linkAddr}> remediate and resolve the issue.</Link>
            </div>
        </div>
    );
}

Explanation.propTypes = {
    policyId: PropTypes.string.isRequired,
    lifecycleStage: PropTypes.string.isRequired
};

export default Explanation;
