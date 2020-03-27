import React from 'react';
import PropTypes from 'prop-types';

import ANALYST_NOTES_TYPES from 'constants/analystnotes';
import AnalystTags from 'Containers/AnalystNotes/AnalystTags';

const ProcessTags = ({ deploymentID, containerName, execFilePath, args }) => {
    const variables = { key: { deploymentID, containerName, execFilePath, args } };
    return <AnalystTags type={ANALYST_NOTES_TYPES.PROCESS} variables={variables} />;
};

ProcessTags.propTypes = {
    deploymentID: PropTypes.string.isRequired,
    containerName: PropTypes.string.isRequired,
    execFilePath: PropTypes.string.isRequired,
    args: PropTypes.string.isRequired
};

export default ProcessTags;
