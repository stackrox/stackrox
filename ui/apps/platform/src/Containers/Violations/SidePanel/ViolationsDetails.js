import React from 'react';
import PropTypes from 'prop-types';

import ViolationComments from 'Containers/AnalystNotes/ViolationComments';
import ViolationTags from 'Containers/AnalystNotes/ViolationTags';
import DeploytimeMessages from './DeploytimeMessages';
import RuntimeMessages from './RuntimeMessages';

function ViolationsDetails({ violationId, violations, processViolation }) {
    return (
        <div className="w-full px-3 pb-5 mt-5">
            <div className="mb-4" data-testid="violation-tags">
                <ViolationTags resourceId={violationId} />
            </div>
            <div className="mb-4" data-testid="violation-comments">
                <ViolationComments resourceId={violationId} />
            </div>
            <RuntimeMessages processViolation={processViolation} />
            <DeploytimeMessages violations={violations} />
        </div>
    );
}

ViolationsDetails.propTypes = {
    violationId: PropTypes.string.isRequired,
    violations: PropTypes.arrayOf(
        PropTypes.shape({
            message: PropTypes.string.isRequired,
        })
    ),
    processViolation: PropTypes.shape({
        message: PropTypes.string.isRequired,
        processes: PropTypes.arrayOf(
            PropTypes.shape({
                id: PropTypes.string.isRequired,
            })
        ).isRequired,
    }),
};

ViolationsDetails.defaultProps = {
    violations: [],
    processViolation: null,
};

export default ViolationsDetails;
