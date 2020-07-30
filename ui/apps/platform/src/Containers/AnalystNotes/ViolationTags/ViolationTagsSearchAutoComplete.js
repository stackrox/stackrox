import React from 'react';
import PropTypes from 'prop-types';

import {
    violationCategories,
    getViolationQuery,
} from 'Containers/AnalystNotes/analystNotesUtils/tagsAutoCompleteVariables';
import TagsSearchAutoComplete from 'Containers/AnalystNotes/TagsSearchAutoComplete';

const ViolationsTagsSearchAutoComplete = ({ children }) => {
    return (
        <TagsSearchAutoComplete
            categories={violationCategories}
            getQueryWithAutoComplete={getViolationQuery}
        >
            {({ isLoading, options, onInputChange, autoCompleteVariables }) => {
                return children({ isLoading, options, onInputChange, autoCompleteVariables });
            }}
        </TagsSearchAutoComplete>
    );
};

ViolationsTagsSearchAutoComplete.propTypes = {
    children: PropTypes.oneOfType([PropTypes.arrayOf(PropTypes.node), PropTypes.node]).isRequired,
};

export default ViolationsTagsSearchAutoComplete;
