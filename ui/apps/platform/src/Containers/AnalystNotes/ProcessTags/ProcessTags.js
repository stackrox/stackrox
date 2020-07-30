import React from 'react';

import ANALYST_NOTES_TYPES from 'constants/analystnotes';
import AnalystTags from 'Containers/AnalystNotes/AnalystTags';
import ProcessTagsSearchAutoComplete from './ProcessTagsSearchAutoComplete';
import ProcessKeyProps from '../processKeyProps';

const ProcessTags = ({ deploymentID, containerName, execFilePath, args }) => {
    const variables = { key: { deploymentID, containerName, execFilePath, args } };

    return (
        <div data-testid="process-tags">
            <ProcessTagsSearchAutoComplete>
                {({ isLoading, options, onInputChange, autoCompleteVariables }) => (
                    <AnalystTags
                        type={ANALYST_NOTES_TYPES.PROCESS}
                        variables={variables}
                        autoComplete={options}
                        autoCompleteVariables={autoCompleteVariables}
                        isLoadingAutoComplete={isLoading}
                        onInputChange={onInputChange}
                    />
                )}
            </ProcessTagsSearchAutoComplete>
        </div>
    );
};

ProcessTags.propTypes = {
    ...ProcessKeyProps,
};

export default ProcessTags;
