import React from 'react';
import PropTypes from 'prop-types';

import ANALYST_NOTES_TYPES from 'constants/analystnotes';
import AnalystComments from 'Containers/AnalystNotes/AnalystComments';

const ViolationComments = ({ resourceId }) => {
    const variables = { resourceId };
    return <AnalystComments type={ANALYST_NOTES_TYPES.VIOLATION} variables={variables} />;
};

ViolationComments.propTypes = {
    resourceId: PropTypes.string.isRequired
};

export default ViolationComments;
