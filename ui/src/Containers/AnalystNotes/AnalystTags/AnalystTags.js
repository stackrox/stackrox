import React from 'react';
import PropTypes from 'prop-types';
import { useQuery, useMutation } from 'react-apollo';
import difference from 'lodash/difference';
import pluralize from 'pluralize';

import ANALYST_NOTES_TYPES from 'constants/analystnotes';
import captureGraphQLErrors from 'utils/captureGraphQLErrors';
import analystNotesLabels from 'messages/analystnotes';
import Tags from 'Components/Tags';
import Message from 'Components/Message';
import SEARCH_AUTOCOMPLETE_QUERY from 'queries/searchAutocomplete';
import { getQueriesByType, getTagsDataByType } from './analystTagsQueries';
import getRefetchQueriesByCondition from '../analystNotesUtils/getRefetchQueriesByCondition';
import GET_PROCESS_COMMENTS_TAGS_COUNT from '../processCommentsTagsQuery';

const AnalystTags = ({
    type,
    variables,
    autoComplete,
    autoCompleteVariables,
    isCollapsible,
    onInputChange,
}) => {
    const { GET_TAGS, ADD_TAGS, REMOVE_TAGS } = getQueriesByType(type);

    const { loading: isLoading, error, data } = useQuery(GET_TAGS, {
        variables,
    });

    // resolves once the modification + refetching happens
    const refetchAndWait = getRefetchQueriesByCondition([
        { query: GET_TAGS, variables },
        {
            query: GET_PROCESS_COMMENTS_TAGS_COUNT,
            variables,
            exclude: type !== ANALYST_NOTES_TYPES.PROCESS,
        },
        {
            query: SEARCH_AUTOCOMPLETE_QUERY,
            variables: autoCompleteVariables,
        },
    ]);

    const [addTags, { loading: isWaitingToAddTags, error: errorOnAddTags }] = useMutation(
        ADD_TAGS,
        refetchAndWait
    );
    const [removeTags, { loading: isWaitingToRemoveTags, error: errorOnRemoveTags }] = useMutation(
        REMOVE_TAGS,
        refetchAndWait
    );

    const { hasErrors } = captureGraphQLErrors([error, errorOnAddTags, errorOnRemoveTags]);

    if (hasErrors)
        return (
            <Message
                type="error"
                message="There was an issue retrieving and/or modifying tags. Please try to view this page again in a little while"
            />
        );

    // disable input when waiting for any sort of modification
    const isDisabled = isWaitingToAddTags || isWaitingToRemoveTags || false;

    const tags = getTagsDataByType(type, data);

    const title = `${tags.length} ${analystNotesLabels[type]} ${pluralize('Tag', tags.length)}`;

    function onChange(updatedTags) {
        const removedTags = difference(tags, updatedTags);
        const addedTags = difference(updatedTags, tags);
        if (addedTags.length)
            addTags({
                variables: { ...variables, tags: addedTags },
            });
        if (removedTags.length)
            removeTags({
                variables: { ...variables, tags: removedTags },
            });
    }

    return (
        <Tags
            title={title}
            tags={tags}
            onChange={onChange}
            onInputChange={onInputChange}
            isLoading={isLoading}
            isDisabled={isDisabled}
            isCollapsible={isCollapsible}
            defaultOpen
            autoComplete={autoComplete}
        />
    );
};

AnalystTags.propTypes = {
    type: PropTypes.string.isRequired,
    variables: PropTypes.shape({}).isRequired,
    autoComplete: PropTypes.arrayOf(PropTypes.string),
    autoCompleteVariables: PropTypes.shape({}).isRequired,
    isCollapsible: PropTypes.bool,
    onInputChange: PropTypes.func.isRequired,
};

AnalystTags.defaultProps = {
    autoComplete: [],
    isCollapsible: true,
};

export default React.memo(AnalystTags);
