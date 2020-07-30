import React from 'react';

import ANALYST_NOTES_TYPES from 'constants/analystnotes';
import AnalystComments from 'Containers/AnalystNotes/AnalystComments';
import ProcessKeyProps from './processKeyProps';

const ProcessComments = ({ deploymentID, containerName, execFilePath, args }) => {
    const variables = { key: { deploymentID, containerName, execFilePath, args } };
    return (
        <div data-testid="process-comments">
            <AnalystComments type={ANALYST_NOTES_TYPES.PROCESS} variables={variables} />
        </div>
    );
};

ProcessComments.propTypes = {
    ...ProcessKeyProps,
};

export default ProcessComments;
