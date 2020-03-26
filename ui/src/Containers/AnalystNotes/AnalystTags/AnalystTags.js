import React from 'react';
import PropTypes from 'prop-types';
import { useQuery, useMutation } from 'react-apollo';
import difference from 'lodash/difference';

import captureGraphQLErrors from 'modules/captureGraphQLErrors';
import analystNotesLabels from 'messages/analystnotes';
import Tags from 'Components/Tags';
import Message from 'Components/Message';
import { getQueriesByType, getTagsDataByType } from './analystTagsQueries';

const AnalystTags = ({ className, type, variables }) => {
    const { GET_TAGS, ADD_TAGS, REMOVE_TAGS } = getQueriesByType(type);

    const { loading: isLoading, error, data } = useQuery(GET_TAGS, {
        variables
    });

    // resolves once the modification + refetching happens
    const refetchAndWait = {
        refetchQueries: () => [{ query: GET_TAGS, variables }],
        awaitRefetchQueries: true
    };

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

    function onChange(updatedTags) {
        const removedTags = difference(tags, updatedTags);
        const addedTags = difference(updatedTags, tags);
        if (addedTags.length)
            addTags({
                variables: { ...variables, tags: addedTags }
            });
        if (removedTags.length)
            removeTags({
                variables: { ...variables, tags: removedTags }
            });
    }

    return (
        <Tags
            className={className}
            label={analystNotesLabels[type]}
            tags={tags}
            onChange={onChange}
            isLoading={isLoading}
            isDisabled={isDisabled}
            defaultOpen
        />
    );
};

AnalystTags.propTypes = {
    type: PropTypes.string.isRequired,
    className: PropTypes.string,
    variables: PropTypes.shape({}).isRequired
};

AnalystTags.defaultProps = {
    className: 'border border-base-400'
};

export default React.memo(AnalystTags);
