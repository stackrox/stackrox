import React from 'react';

import ANALYST_NOTES_TYPES from 'constants/analystnotes';
import AnalystTags from 'Containers/AnalystNotes/AnalystTags';
import ProcessKeyProps from './processKeyProps';

const ProcessTags = ({ deploymentID, containerName, execFilePath, args }) => {
    const variables = { key: { deploymentID, containerName, execFilePath, args } };
    return <AnalystTags type={ANALYST_NOTES_TYPES.PROCESS} variables={variables} />;
};

ProcessTags.propTypes = {
    ...ProcessKeyProps
};

export default ProcessTags;
