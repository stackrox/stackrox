import React from 'react';
import PropTypes from 'prop-types';

import ANALYST_NOTES_TYPES from 'constants/analystnotes';
import AnalystComments from 'Containers/AnalystNotes/AnalystComments';

const ViolationComments = ({ resourceId, isCollapsible }) => {
    const variables = { resourceId };
    return (
        <AnalystComments
            type={ANALYST_NOTES_TYPES.VIOLATION}
            variables={variables}
            isCollapsible={isCollapsible}
        />
    );
};

ViolationComments.propTypes = {
    resourceId: PropTypes.string.isRequired,
    isCollapsible: PropTypes.bool
};

ViolationComments.defaultProps = {
    isCollapsible: true
};

export default ViolationComments;
