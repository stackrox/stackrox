import React from 'react';
import PropTypes from 'prop-types';
import gql from 'graphql-tag';
import { useQuery, useMutation } from 'react-apollo';
import Raven from 'raven-js';

import analystNotesLabels from 'messages/analystnotes';
import CommentThread from 'Components/CommentThread';

const GET_ALERT_COMMENTS = gql`
    query getAlertComments($resourceId: ID!) {
        comments: alertComments(resourceId: $resourceId) {
            resourceType
            resourceId
            user {
                email
                id
                name
            }
            id: commentId
            message: commentMessage
            createdTime: createdAt
            updatedTime: lastModified
            isModifiable: modifiable
        }
    }
`;

const ADD_ALERT_COMMENT = gql`
    mutation addAlertComment($resourceId: ID!, $commentMessage: String!) {
        addAlertComment(resourceId: $resourceId, commentMessage: $commentMessage)
    }
`;

const UPDATE_ALERT_COMMENT = gql`
    mutation updateAlertComment($resourceId: ID!, $commentId: ID!, $commentMessage: String!) {
        updateAlertComment(
            resourceId: $resourceId
            commentId: $commentId
            commentMessage: $commentMessage
        )
    }
`;

const REMOVE_ALERT_COMMENT = gql`
    mutation removeAlertComment($resourceId: ID!, $commentId: ID!) {
        removeAlertComment(resourceId: $resourceId, commentId: $commentId)
    }
`;

const AnalystComments = ({ className, type, id }) => {
    const { loading, error, data } = useQuery(GET_ALERT_COMMENTS, {
        variables: { resourceId: id }
    });

    if (error) Raven.captureException(error);

    const refetchQueries = () => [{ query: GET_ALERT_COMMENTS, variables: { resourceId: id } }];

    const [addComment] = useMutation(ADD_ALERT_COMMENT, { refetchQueries });
    const [updateComment] = useMutation(UPDATE_ALERT_COMMENT, { refetchQueries });
    const [removeComment] = useMutation(REMOVE_ALERT_COMMENT, { refetchQueries });

    const comments = data ? data.comments : [];

    function onCreate(commentMessage) {
        addComment({
            variables: { resourceId: id, commentMessage }
        });
    }

    function onUpdate(commentId, commentMessage) {
        updateComment({
            variables: { resourceId: id, commentId, commentMessage }
        });
    }

    function onRemove(commentId) {
        removeComment({
            variables: { resourceId: id, commentId }
        });
    }

    return (
        <CommentThread
            className={className}
            label={analystNotesLabels[type]}
            comments={comments}
            onCreate={onCreate}
            onUpdate={onUpdate}
            onRemove={onRemove}
            isLoading={loading}
            defaultOpen
        />
    );
};

AnalystComments.propTypes = {
    id: PropTypes.string.isRequired,
    type: PropTypes.string.isRequired,
    className: PropTypes.string
};

AnalystComments.defaultProps = {
    className: 'border border-base-400'
};

export default React.memo(AnalystComments);
