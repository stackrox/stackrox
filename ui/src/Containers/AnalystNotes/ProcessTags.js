import React from 'react';

import ANALYST_NOTES_TYPES from 'constants/analystnotes';
import AnalystTags from 'Containers/AnalystNotes/AnalystTags';
import SearchAutoComplete from 'Containers/Search/SearchAutoComplete';
import ProcessKeyProps from './processKeyProps';

const ProcessTags = ({ deploymentID, containerName, execFilePath, args }) => {
    const variables = { key: { deploymentID, containerName, execFilePath, args } };
    const autoCompleteVariables = { categories: ['DEPLOYMENTS'], query: 'Process Tag:' };
    return (
        <SearchAutoComplete
            categories={autoCompleteVariables.categories}
            query={autoCompleteVariables.query}
        >
            {({ isLoading, options }) => (
                <AnalystTags
                    type={ANALYST_NOTES_TYPES.PROCESS}
                    variables={variables}
                    autoComplete={options}
                    autoCompleteVariables={autoCompleteVariables}
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
