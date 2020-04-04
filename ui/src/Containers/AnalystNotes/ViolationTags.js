import React from 'react';
import PropTypes from 'prop-types';

import ANALYST_NOTES_TYPES from 'constants/analystnotes';
import SearchAutoComplete from 'Containers/Search/SearchAutoComplete';
import AnalystTags from 'Containers/AnalystNotes/AnalystTags';

const ViolationTags = ({ resourceId }) => {
    const variables = { resourceId };
    const autoCompleteVariables = { categories: ['ALERTS'], query: 'Tag:' };
    return (
        <SearchAutoComplete
            categories={autoCompleteVariables.categories}
            query={autoCompleteVariables.query}
        >
            {({ isLoading, options }) => (
                <AnalystTags
                    type={ANALYST_NOTES_TYPES.VIOLATION}
                    variables={variables}
                    autoComplete={options}
                    autoCompleteVariables={autoCompleteVariables}
                    isLoadingAutoComplete={isLoading}
                />
            )}
        </SearchAutoComplete>
    );
};

ViolationTags.propTypes = {
    resourceId: PropTypes.string.isRequired
};

export default ViolationTags;
