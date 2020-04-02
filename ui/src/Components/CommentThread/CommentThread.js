import React, { useState } from 'react';
import PropTypes from 'prop-types';
import pluralize from 'pluralize';
import sortBy from 'lodash/sortBy';

import CollapsibleCard from 'Components/CollapsibleCard';
import Button from 'Components/Button';
import { PlusCircle } from 'react-feather';
import NoResultsMessage from 'Components/NoResultsMessage';
import Loader from 'Components/Loader';
import Comment from './Comment';

const CommentThread = ({
    className,
    label,
    comments,
    onCreate,
    onUpdate,
    onRemove,
    defaultLimit,
    defaultOpen,
    isLoading,
    isDisabled
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
            message: ''
        });
    }

    function onClose() {
        createComment(null);
    }

    function onSave(id, message) {
        if (!id) onCreate(message);
        else onUpdate(id, message);
    }

    let content = (
        <div className="p-3">
            <Loader />
        </div>
    );
    if (!isLoading) {
        content =
            comments.length > 0 || !!newComment ? (
                <div className="p-3">
                    {sortedComments.slice(0, limit).map((comment, i) => (
                        <div key={comment.id} className={i === 0 ? 'mt-0' : 'mt-3'}>
                            <Comment
                                comment={comment}
                                onSave={onSave}
                                onRemove={onRemove}
                                isDisabled={isDisabled}
                            />
                        </div>
                    ))}
                    {!!newComment && (
                        <div className={sortedComments.length === 0 ? 'mt-0' : 'mt-3'}>
                            <Comment
                                comment={newComment}
                                onSave={onSave}
                                onClose={onClose}
                                onRemove={onRemove}
                                defaultEdit
                                isDisabled={isDisabled}
                            />
                        </div>
                    )}
                    {hasMoreComments && (
                        <div className="flex flex-1 justify-center mt-3">
                            <Button
                                className="bg-primary-200 border border-primary-800 hover:bg-primary-300 p-1 rounded-full rounded-sm text-sm text-success-900 uppercase"
                                text="Load More Comments"
                                onClick={showMoreComments}
                                disabled={isDisabled}
                            />
                        </div>
                    )}
                </div>
            ) : (
                <div className="p-4">
                    <NoResultsMessage message="No Comments" />
                </div>
            );
    }

    return (
        <CollapsibleCard
            cardClassName={className}
            title={`${length} ${label} ${pluralize('Comment', length)}`}
            headerComponents={
                <Button
                    className="bg-primary-200 border border-primary-800 hover:bg-primary-300 p-1 rounded-sm text-sm text-success-900 uppercase"
                    text="New"
                    icon={<PlusCircle className="text-primary-800 h-4 w-4 mr-1" />}
                    disabled={!!newComment || isDisabled}
                    onClick={addNewComment}
                />
            }
            open={defaultOpen}
        >
            {content}
        </CollapsibleCard>
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
                email: PropTypes.string.isRequired
            }),
            createdTime: PropTypes.string.isRequired,
            updatedTime: PropTypes.string.isRequired,
            isEditable: PropTypes.bool,
            isDeletable: PropTypes.bool
        })
    ),
    onCreate: PropTypes.func.isRequired,
    onUpdate: PropTypes.func.isRequired,
    onRemove: PropTypes.func.isRequired,
    defaultLimit: PropTypes.number,
    defaultOpen: PropTypes.bool,
    className: PropTypes.string,
    isLoading: PropTypes.bool,
    isDisabled: PropTypes.bool
};

CommentThread.defaultProps = {
    comments: [],
    defaultLimit: 5,
    defaultOpen: false,
    className: 'border border-base-400',
    isLoading: false,
    isDisabled: false
};

export default CommentThread;
