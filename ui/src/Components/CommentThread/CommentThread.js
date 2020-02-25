import React, { useState } from 'react';
import PropTypes from 'prop-types';
import pluralize from 'pluralize';
import sortBy from 'lodash/sortBy';

import CollapsibleCard from 'Components/CollapsibleCard';
import Button from 'Components/Button';
import { PlusCircle } from 'react-feather';
import NoResultsMessage from 'Components/NoResultsMessage';
import Comment from './Comment';

const CommentThread = ({
    className,
    type,
    currentUser,
    comments,
    onSave,
    onDelete,
    defaultLimit,
    defaultOpen
}) => {
    const [newComment, createComment] = useState(null);
    const [limit, setLimit] = useState(defaultLimit);

    const sortedComments = sortBy(comments, ['createdTime', 'user']);
    const { length } = sortedComments;
    const hasMoreComments = limit < length;

    function showMoreComments() {
        setLimit(limit + defaultLimit);
    }

    function addNewComment(e) {
        e.stopPropagation(); // prevents click-through trigger of collapsible
        createComment({
            user: currentUser,
            createdTime: new Date().toISOString(),
            message: '',
            canModify: true
        });
    }

    function onClose() {
        createComment(null);
    }

    const content =
        comments.length > 0 || !!newComment ? (
            <div className="p-3">
                {sortedComments.slice(0, limit).map((comment, i) => (
                    <div key={comment.id} className={i === 0 ? 'mt-0' : 'mt-3'}>
                        <Comment comment={comment} onSave={onSave} onDelete={onDelete} />
                    </div>
                ))}
                {!!newComment && (
                    <div className={sortedComments.length === 0 ? 'mt-0' : 'mt-3'}>
                        <Comment
                            comment={newComment}
                            onSave={onSave}
                            onClose={onClose}
                            onDelete={onDelete}
                            defaultEdit
                        />
                    </div>
                )}
                {hasMoreComments && (
                    <div className="flex flex-1 justify-center mt-3">
                        <Button
                            className="bg-primary-200 border border-primary-800 hover:bg-primary-300 p-1 rounded-full rounded-sm text-sm text-success-900 uppercase"
                            text="Load More Comments"
                            onClick={showMoreComments}
                        />
                    </div>
                )}
            </div>
        ) : (
            <div className="p-4">
                <NoResultsMessage message="No Comments Available" />
            </div>
        );

    return (
        <CollapsibleCard
            cardClassName={className}
            title={`${length} ${type} ${pluralize('Comment', length)}`}
            headerComponents={
                <Button
                    className="bg-primary-200 border border-primary-800 hover:bg-primary-300 p-1 rounded-sm text-sm text-success-900 uppercase"
                    text="New"
                    icon={<PlusCircle className="text-primary-800 h-4 w-4 mr-1" />}
                    disabled={!!newComment}
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
    type: PropTypes.string.isRequired,
    currentUser: PropTypes.string.isRequired,
    comments: PropTypes.arrayOf(
        PropTypes.shape({
            id: PropTypes.string,
            message: PropTypes.string,
            user: PropTypes.string,
            createdTime: PropTypes.string,
            updatedTime: PropTypes.string,
            canModify: PropTypes.bool
        })
    ),
    onSave: PropTypes.func.isRequired,
    onDelete: PropTypes.func.isRequired,
    defaultLimit: PropTypes.number,
    defaultOpen: PropTypes.bool,
    className: PropTypes.string
};

CommentThread.defaultProps = {
    comments: [],
    defaultLimit: 3,
    defaultOpen: false,
    className: 'border border-base-400'
};

export default CommentThread;
