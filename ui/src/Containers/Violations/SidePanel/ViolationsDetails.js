import React from 'react';
import PropTypes from 'prop-types';

import DeploytimeMessages from './DeploytimeMessages';
import RuntimeMessages from './RuntimeMessages';

function ViolationsDetails({ violations, processViolation }) {
    return (
        <div className="w-full px-3 pb-5 mt-5">
            <RuntimeMessages processViolation={processViolation} />
            <DeploytimeMessages violations={violations} />
        </div>
    );
}

ViolationsDetails.propTypes = {
    violations: PropTypes.arrayOf(
        PropTypes.shape({
            message: PropTypes.string.isRequired
        })
    ),
    processViolation: PropTypes.shape({
        message: PropTypes.string.isRequired,
        processes: PropTypes.array.isRequired
    })
};

ViolationsDetails.defaultProps = {
    violations: [],
    processViolation: null
};

export default ViolationsDetails;
