import React from 'react';
import PropTypes from 'prop-types';
import { useQuery, useMutation } from 'react-apollo';

import ANALYST_NOTES_TYPES from 'constants/analystnotes';
import captureGraphQLErrors from 'modules/captureGraphQLErrors';
import analystNotesLabels from 'messages/analystnotes';
import Message from 'Components/Message';
import CommentThread from 'Components/CommentThread';
import { getQueriesByType } from './analystCommentsQueries';
import getRefetchQueriesByCondition from '../analystNotesUtils/getRefetchQueriesByCondition';
import GET_PROCESS_COMMENTS_TAGS_COUNT from '../processCommentsTagsQuery';

// the prop "variables" is an object with the necessary variables for querying the comments APIs
const AnalystComments = ({ type, variables, isCollapsible }) => {
    const { GET_COMMENTS, ADD_COMMENT, UPDATE_COMMENT, REMOVE_COMMENT } = getQueriesByType(type);

    const { loading: isLoading, error, data } = useQuery(GET_COMMENTS, {
        variables
    });

    // resolves once the modification + refetching happens
    const refetchAndWait = getRefetchQueriesByCondition([
        { query: GET_COMMENTS, variables, exclude: false },
        {
            query: GET_PROCESS_COMMENTS_TAGS_COUNT,
            variables,
            exclude: type !== ANALYST_NOTES_TYPES.PROCESS
        }
    ]);

    const [addComment, { loading: isWaitingToAddComment, error: errorOnAddComment }] = useMutation(
        ADD_COMMENT,
        refetchAndWait
    );
    const [
        updateComment,
        { loading: isWaitingToUpdateComment, error: errorOnUpdateComment }
    ] = useMutation(UPDATE_COMMENT, refetchAndWait);
    const [
        removeComment,
        { loading: isWaitingToRemoveComment, error: errorOnRemoveComment }
    ] = useMutation(REMOVE_COMMENT, refetchAndWait);

    const { hasErrors } = captureGraphQLErrors([
        error,
        errorOnAddComment,
        errorOnUpdateComment,
        errorOnRemoveComment
    ]);

    if (hasErrors)
        return (
            <Message
                type="error"
                message="There was an issue retrieving and/or modifying comments. Please try to view this page again in a little while"
            />
        );

    // disable buttons/inputs when waiting for any sort of modification
    const isDisabled =
        isWaitingToAddComment || isWaitingToUpdateComment || isWaitingToRemoveComment || false;

    const comments = data ? data.comments : [];

    function onCreate(commentMessage) {
        addComment({
            variables: { ...variables, commentMessage }
        });
    }

    function onUpdate(commentId, commentMessage) {
        updateComment({
            variables: { ...variables, commentId, commentMessage }
        });
    }

    function onRemove(commentId) {
        removeComment({
            variables: { ...variables, commentId }
        });
    }

    return (
        <CommentThread
            label={analystNotesLabels[type]}
            comments={comments}
            onCreate={onCreate}
            onUpdate={onUpdate}
            onRemove={onRemove}
            isLoading={isLoading}
            isDisabled={isDisabled}
            isCollapsible={isCollapsible}
            defaultOpen
        />
    );
};

AnalystComments.propTypes = {
    type: PropTypes.string.isRequired,
    variables: PropTypes.shape({}).isRequired,
    isCollapsible: PropTypes.bool
};

AnalystComments.defaultProps = {
    isCollapsible: true
};

export default React.memo(AnalystComments);
