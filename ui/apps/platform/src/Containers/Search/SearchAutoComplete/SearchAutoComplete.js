import React from 'react';
import PropTypes from 'prop-types';
import { useQuery } from '@apollo/client';
import { Message } from '@stackrox/ui-components';

import SEARCH_AUTOCOMPLETE_QUERY from 'queries/searchAutocomplete';
import captureGraphQLErrors from 'utils/captureGraphQLErrors';

const SearchAutoComplete = ({ categories, query, children }) => {
    const {
        loading: isLoading,
        error,
        data,
    } = useQuery(SEARCH_AUTOCOMPLETE_QUERY, {
        variables: {
            categories,
            query,
        },
    });

    const { hasErrors } = captureGraphQLErrors([error]);

    if (hasErrors) {
        return (
            <Message type="error">
                There was an issue retrieving autocomplete options. Please try to view this page
                again.
            </Message>
        );
    }

    const options = data && data.searchAutocomplete;

    return children({ isLoading, options });
};

SearchAutoComplete.propTypes = {
    categories: PropTypes.arrayOf(PropTypes.string).isRequired,
    query: PropTypes.string,
    children: PropTypes.func.isRequired,
};

SearchAutoComplete.defaultProps = {
    query: '',
};

export default SearchAutoComplete;
