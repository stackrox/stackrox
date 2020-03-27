import React from 'react';
import PropTypes from 'prop-types';

import ANALYST_NOTES_TYPES from 'constants/analystnotes';
import AnalystComments from 'Containers/AnalystNotes/AnalystComments';

const ProcessComments = ({ deploymentID, containerName, execFilePath, args }) => {
    const variables = { key: { deploymentID, containerName, execFilePath, args } };
    return <AnalystComments type={ANALYST_NOTES_TYPES.PROCESS} variables={variables} />;
};

ProcessComments.propTypes = {
    deploymentID: PropTypes.string.isRequired,
    containerName: PropTypes.string.isRequired,
    execFilePath: PropTypes.string.isRequired,
    args: PropTypes.string.isRequired
};

export default ProcessComments;
