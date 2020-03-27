import React from 'react';
import PropTypes from 'prop-types';

import { knownBackendFlags } from 'utils/featureFlags';
import FeatureEnabled from 'Containers/FeatureEnabled';
import ViolationComments from 'Containers/AnalystNotes/ViolationComments';
import ViolationTags from 'Containers/AnalystNotes/ViolationTags';
import DeploytimeMessages from './DeploytimeMessages';
import RuntimeMessages from './RuntimeMessages';

function ViolationsDetails({ violationId, violations, processViolation }) {
    return (
        <div className="w-full px-3 pb-5 mt-5">
            <FeatureEnabled featureFlag={knownBackendFlags.ROX_ANALYST_NOTES_UI}>
                <div className="mb-4">
                    <ViolationTags resourceId={violationId} />
                </div>
                <div className="mb-4">
                    <ViolationComments resourceId={violationId} />
                </div>
            </FeatureEnabled>
            <RuntimeMessages processViolation={processViolation} />
            <DeploytimeMessages violations={violations} />
        </div>
    );
}

ViolationsDetails.propTypes = {
    violationId: PropTypes.string.isRequired,
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
