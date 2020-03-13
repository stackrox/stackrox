import React from 'react';
import PropTypes from 'prop-types';
import gql from 'graphql-tag';
import { useQuery, useMutation } from 'react-apollo';
import difference from 'lodash/difference';

import captureGraphQLErrors from 'modules/captureGraphQLErrors';
import analystNotesLabels from 'messages/analystnotes';
import Tags from 'Components/Tags';
import Message from 'Components/Message';

const GET_ALERT_TAGS = gql`
    query getAlertTags($resourceId: ID!) {
        violation(id: $resourceId) {
            id
            tags
        }
    }
`;

const ADD_ALERT_TAGS = gql`
    mutation addAlertTags($resourceId: ID!, $tags: [String!]!) {
        addAlertTags(resourceId: $resourceId, tags: $tags)
    }
`;

const REMOVE_ALERT_TAGS = gql`
    mutation removeAlertTags($resourceId: ID!, $tags: [String!]!) {
        removeAlertTags(resourceId: $resourceId, tags: $tags)
    }
`;

const AnalystTags = ({ className, type, id }) => {
    const { loading, error, data } = useQuery(GET_ALERT_TAGS, {
        variables: { resourceId: id }
    });

    const refetchQueries = () => [{ query: GET_ALERT_TAGS, variables: { resourceId: id } }];

    const [addTags, { loading: waitingToAddTags, error: errorOnAddTags }] = useMutation(
        ADD_ALERT_TAGS,
        { refetchQueries, awaitRefetchQueries: true }
    );
    const [removeTags, { loading: waitingToRemoveTags, error: errorOnRemoveTags }] = useMutation(
        REMOVE_ALERT_TAGS,
        { refetchQueries, awaitRefetchQueries: true }
    );

    const { hasErrors } = captureGraphQLErrors([error, errorOnAddTags, errorOnRemoveTags]);
    if (hasErrors)
        return (
            <Message
                type="error"
                message="There was an issue retrieving and/or modifying tags for this violation. Please try to view this page again in a little while"
            />
        );

    const isDisabled = waitingToAddTags || waitingToRemoveTags || false;

    const tags = data && data.violation.tags ? data.violation.tags : [];

    function onChange(newTags) {
        const removedTags = difference(tags, newTags);
        const addedTags = difference(newTags, tags);
        addTags({
            variables: { resourceId: id, tags: addedTags }
        });
        removeTags({
            variables: { resourceId: id, tags: removedTags }
        });
    }

    return (
        <Tags
            className={className}
            label={analystNotesLabels[type]}
            tags={tags}
            onChange={onChange}
            isLoading={loading}
            isDisabled={isDisabled}
            defaultOpen
        />
    );
};

AnalystTags.propTypes = {
    type: PropTypes.string.isRequired,
    id: PropTypes.string.isRequired,
    className: PropTypes.string
};

AnalystTags.defaultProps = {
    className: 'border border-base-400'
};

export default React.memo(AnalystTags);
