import React from 'react';
import PropTypes from 'prop-types';
import { useQuery } from 'react-apollo';

import SEARCH_AUTOCOMPLETE_QUERY from 'queries/searchAutocomplete';
import captureGraphQLErrors from 'utils/captureGraphQLErrors';
import Message from 'Components/Message';

const SearchAutoComplete = ({ categories, query, children }) => {
    const { loading: isLoading, error, data } = useQuery(SEARCH_AUTOCOMPLETE_QUERY, {
        variables: {
            categories,
            query,
        },
    });

    const { hasErrors } = captureGraphQLErrors([error]);

    if (hasErrors)
        return (
            <Message
                type="error"
                message="There was an issue retrieving autocomplete options. Please try to view this page again."
            />
        );

    const options = data && data.searchAutocomplete;

    return children({ isLoading, options });
};

SearchAutoComplete.propTypes = {
    categories: PropTypes.arrayOf(PropTypes.string).isRequired,
    query: PropTypes.string,
};

SearchAutoComplete.defaultProps = {
    query: '',
};

export default SearchAutoComplete;
