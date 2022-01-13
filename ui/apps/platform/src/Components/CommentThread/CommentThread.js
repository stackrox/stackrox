import React, { useState } from 'react';
import PropTypes from 'prop-types';
import pluralize from 'pluralize';
import sortBy from 'lodash/sortBy';
import {
    Card,
    CardHeader,
    CardBody,
    CardActions,
    Button,
    Bullseye,
    Spinner,
    Flex,
    FlexItem,
    EmptyState,
} from '@patternfly/react-core';
import { PlusCircleIcon } from '@patternfly/react-icons';

import Comment from './Comment';

const CommentThread = ({
    label,
    comments,
    onCreate,
    onUpdate,
    onRemove,
    defaultLimit,
    isLoading,
    isDisabled,
}) => {
    const [newComment, createComment] = useState(null);
    const [limit, setLimit] = useState(defaultLimit);

    const sortedComments = sortBy(comments, ['createdTime']);
    const { length } = sortedComments;
    const hasMoreComments = limit < length;

    function showMoreComments() {
        setLimit(limit + defaultLimit);
    }

    function addNewComment(e) {
        e.stopPropagation(); // prevents click-through trigger of collapsible
        createComment({
            createdTime: new Date().toISOString(),
            message: '',
        });
    }

    function onClose() {
        createComment(null);
    }

    function onSave(id, message) {
        if (!id) {
            onCreate(message);
        } else {
            onUpdate(id, message);
        }
    }

    let content = (
        <Bullseye>
            <Spinner isSVG />
        </Bullseye>
    );
    if (!isLoading) {
        content =
            comments.length > 0 || !!newComment ? (
                <Flex direction={{ default: 'column' }}>
                    {sortedComments.slice(0, limit).map((comment) => (
                        <FlexItem key={comment.id} data-testid="comment">
                            <Comment
                                comment={comment}
                                onSave={onSave}
                                onRemove={onRemove}
                                isDisabled={isDisabled}
                            />
                        </FlexItem>
                    ))}
                    {!!newComment && (
                        <FlexItem data-testid="new-comment">
                            <Comment
                                comment={newComment}
                                onSave={onSave}
                                onClose={onClose}
                                onRemove={onRemove}
                                defaultEdit
                                isDisabled={isDisabled}
                            />
                        </FlexItem>
                    )}
                    {hasMoreComments && (
                        <FlexItem alignSelf={{ default: 'alignSelfCenter' }}>
                            <Button onClick={showMoreComments} isDisabled={isDisabled}>
                                Load More Comments
                            </Button>
                        </FlexItem>
                    )}
                </Flex>
            ) : (
                <EmptyState>No Comments</EmptyState>
            );
    }

    return (
        <Card isFlat>
            <CardHeader>
                {`${length} ${label} ${pluralize('Comment', length)}`}
                <CardActions>
                    <Button
                        variant="secondary"
                        icon={<PlusCircleIcon />}
                        disabled={!!newComment || isDisabled}
                        onClick={addNewComment}
                        data-testid="new-comment-button"
                    >
                        New
                    </Button>
                </CardActions>
            </CardHeader>
            <CardBody>{content}</CardBody>
        </Card>
    );
};

CommentThread.propTypes = {
    label: PropTypes.string.isRequired,
    comments: PropTypes.arrayOf(
        PropTypes.shape({
            id: PropTypes.string.isRequired,
            message: PropTypes.string,
            user: PropTypes.shape({
                id: PropTypes.string.isRequired,
                name: PropTypes.string.isRequired,
                email: PropTypes.string.isRequired,
            }),
            createdTime: PropTypes.string.isRequired,
            updatedTime: PropTypes.string.isRequired,
            isEditable: PropTypes.bool,
            isDeletable: PropTypes.bool,
        })
    ),
    onCreate: PropTypes.func.isRequired,
    onUpdate: PropTypes.func.isRequired,
    onRemove: PropTypes.func.isRequired,
    defaultLimit: PropTypes.number,
    isLoading: PropTypes.bool,
    isDisabled: PropTypes.bool,
};

CommentThread.defaultProps = {
    comments: [],
    defaultLimit: 5,
    isLoading: false,
    isDisabled: false,
};

export default CommentThread;
