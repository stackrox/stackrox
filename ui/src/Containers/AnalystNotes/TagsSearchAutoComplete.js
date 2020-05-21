import React, { useState } from 'react';
import PropTypes from 'prop-types';
import debounce from 'lodash/debounce';

import SearchAutoComplete from 'Containers/Search/SearchAutoComplete';

const TagsSearchAutoComplete = ({ categories, getQueryWithAutoComplete, children }) => {
    const [autoComplete, setAutoComplete] = useState('');

    const onInputChange = debounce(setAutoComplete, 250, { maxWait: 1000 });

    const autoCompleteVariables = {
        categories,
        query: getQueryWithAutoComplete(autoComplete),
    };

    return (
        <SearchAutoComplete
            categories={autoCompleteVariables.categories}
            query={autoCompleteVariables.query}
        >
            {({ isLoading, options }) => {
                return children({ isLoading, options, onInputChange, autoCompleteVariables });
            }}
        </SearchAutoComplete>
    );
};

TagsSearchAutoComplete.propTypes = {
    categories: PropTypes.arrayOf(PropTypes.string).isRequired,
    getQueryWithAutoComplete: PropTypes.func.isRequired,
    children: PropTypes.func.isRequired,
};

export default TagsSearchAutoComplete;
