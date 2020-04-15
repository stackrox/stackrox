import React from 'react';
import PropTypes from 'prop-types';

import {
    processCategories,
    getProcessQuery
} from 'Containers/AnalystNotes/analystNotesUtils/tagsAutoCompleteVariables';
import TagsSearchAutoComplete from 'Containers/AnalystNotes/TagsSearchAutoComplete';

const ProcessTagsSearchAutoComplete = ({ children }) => {
    return (
        <TagsSearchAutoComplete
            categories={processCategories}
            getQueryWithAutoComplete={getProcessQuery}
        >
            {({ isLoading, options, onInputChange, autoCompleteVariables }) => {
                return children({ isLoading, options, onInputChange, autoCompleteVariables });
            }}
        </TagsSearchAutoComplete>
    );
};

ProcessTagsSearchAutoComplete.propTypes = {
    children: PropTypes.oneOfType([PropTypes.arrayOf(PropTypes.node), PropTypes.node]).isRequired
};

export default ProcessTagsSearchAutoComplete;
