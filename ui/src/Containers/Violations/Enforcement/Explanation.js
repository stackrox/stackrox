import React from 'react';
import PropTypes from 'prop-types';
import { Link } from 'react-router-dom';

import lifecycleToExplanation from 'Containers/Violations/Enforcement/descriptors';

function Explanation({ listAlert }) {
    if (!listAlert.enforcementCount || listAlert.enforcementCount === 0) {
        return '';
    }

    const linkAddr = `../policies/${listAlert.policy.id}`;
    return (
        <div className="h-full p-3 text-base-600 font-600 text-base leading-loose">
            <div>{lifecycleToExplanation[listAlert.lifecycleStage]}</div>
            <hr />
            <div>
                If the enforcement action is being applied several times, learn more on how you can
                <Link to={linkAddr}> remediate and resolve the issue.</Link>
            </div>
        </div>
    );
}

Explanation.propTypes = {
    listAlert: PropTypes.shape({
        policy: PropTypes.shape({
            id: PropTypes.string.isRequired
        }).isRequired,
        lifecycleStage: PropTypes.string.isRequired,
        enforcementCount: PropTypes.number
    }).isRequired
};

export default Explanation;
