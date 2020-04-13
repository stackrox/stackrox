import React from 'react';

import { processTagsAutoCompleteVariables } from 'Containers/AnalystNotes/analystNotesUtils/tagsAutoCompleteVariables';
import ANALYST_NOTES_TYPES from 'constants/analystnotes';
import AnalystTags from 'Containers/AnalystNotes/AnalystTags';
import SearchAutoComplete from 'Containers/Search/SearchAutoComplete';
import ProcessKeyProps from './processKeyProps';

const ProcessTags = ({ deploymentID, containerName, execFilePath, args }) => {
    const variables = { key: { deploymentID, containerName, execFilePath, args } };
    return (
        <SearchAutoComplete
            categories={processTagsAutoCompleteVariables.categories}
            query={processTagsAutoCompleteVariables.query}
        >
            {({ isLoading, options }) => (
                <AnalystTags
                    type={ANALYST_NOTES_TYPES.PROCESS}
                    variables={variables}
                    autoComplete={options}
                    autoCompleteVariables={processTagsAutoCompleteVariables}
                    isLoadingAutoComplete={isLoading}
                />
            )}
        </SearchAutoComplete>
    );
};

ProcessTags.propTypes = {
    ...ProcessKeyProps
};

export default ProcessTags;
