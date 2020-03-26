import React from 'react';
import PropTypes from 'prop-types';

import ANALYST_NOTES_TYPES from 'constants/analystnotes';
import AnalystTags from 'Containers/AnalystNotes/AnalystTags';

const ViolationTags = ({ resourceId }) => {
    const variables = { resourceId };
    return <AnalystTags type={ANALYST_NOTES_TYPES.VIOLATION} variables={variables} />;
};

ViolationTags.propTypes = {
    resourceId: PropTypes.string.isRequired
};

export default ViolationTags;
