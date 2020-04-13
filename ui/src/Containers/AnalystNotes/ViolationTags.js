import React from 'react';
import PropTypes from 'prop-types';

import { violationTagsAutoCompleteVariables } from 'Containers/AnalystNotes/analystNotesUtils/tagsAutoCompleteVariables';
import ANALYST_NOTES_TYPES from 'constants/analystnotes';
import SearchAutoComplete from 'Containers/Search/SearchAutoComplete';
import AnalystTags from 'Containers/AnalystNotes/AnalystTags';

const ViolationTags = ({ resourceId, isCollapsible }) => {
    const variables = { resourceId };
    return (
        <SearchAutoComplete
            categories={violationTagsAutoCompleteVariables.categories}
            query={violationTagsAutoCompleteVariables.query}
        >
            {({ isLoading, options }) => (
                <AnalystTags
                    type={ANALYST_NOTES_TYPES.VIOLATION}
                    variables={variables}
                    isCollapsible={isCollapsible}
                    autoComplete={options}
                    autoCompleteVariables={violationTagsAutoCompleteVariables}
                    isLoadingAutoComplete={isLoading}
                />
            )}
        </SearchAutoComplete>
    );
};

ViolationTags.propTypes = {
    resourceId: PropTypes.string.isRequired,
    isCollapsible: PropTypes.string
};

ViolationTags.defaultProps = {
    isCollapsible: true
};

export default ViolationTags;
