import React, { useState } from 'react';
import PropTypes from 'prop-types';
import { format } from 'date-fns';

import dateTimeFormat from 'constants/dateTimeFormat';
import CustomDialogue from 'Components/CustomDialogue';
import CommentForm from './CommentForm';
import CommentActionButtons from './CommentActionButtons';
import CommentMessage from './CommentMessage';

const Comment = ({ comment, onRemove, onSave, onClose, defaultEdit, isDisabled }) => {
    const [isEditing, setEdit] = useState(defaultEdit);
    const [isDialogueOpen, setIsDialogueOpen] = useState(false);

    const { id, user, createdTime, updatedTime, message, isDeletable, isEditable } = comment;

    const isCommentUpdated = updatedTime && createdTime !== updatedTime;

    const textHeader = user ? user.name : 'Add New Comment';

    const initialFormValues = { message };

    function onEdit() {
        setEdit(true);
    }

    function onCloseHandler() {
        setEdit(false);
        onClose();
    }

    function onSubmit(data) {
        onCloseHandler();
        onSave(id, data.message);
    }

    function onRemoveHandler() {
        setIsDialogueOpen(true);
    }

    function cancelDeletion() {
        setIsDialogueOpen(false);
    }

    function confirmDeletion() {
        onRemove(id);
        setIsDialogueOpen(false);
    }

    return (
        <div
            className={`${
                isEditing
                    ? 'bg-success-200 border-success-500'
                    : 'bg-primary-100 border-primary-300'
            } border rounded-lg p-2`}
        >
            <div className="flex flex-1">
                <div className="text-primary-800 flex flex-1">{textHeader}</div>
                <CommentActionButtons
                    isEditing={isEditing}
                    isEditable={isEditable}
                    isDeletable={isDeletable}
                    onEdit={onEdit}
                    onRemove={onRemoveHandler}
                    onClose={onCloseHandler}
                    isDisabled={isDisabled}
                />
            </div>
            <div className="text-base-500 text-xs mt-1">
                {createdTime && format(createdTime, dateTimeFormat)}{' '}
                {isCommentUpdated && '(edited)'}
            </div>
            <div className="mt-2 text-primary-800 leading-normal">
                {isEditing ? (
                    <CommentForm initialFormValues={initialFormValues} onSubmit={onSubmit} />
                ) : (
                    <CommentMessage message={message} />
                )}
            </div>
            {isDialogueOpen && (
                <CustomDialogue
                    title="Delete Comment?"
                    onConfirm={confirmDeletion}
                    confirmText="Yes"
                    onCancel={cancelDeletion}
                />
            )}
        </div>
    );
};

Comment.propTypes = {
    comment: PropTypes.shape({
        id: PropTypes.string,
        message: PropTypes.string,
        user: PropTypes.shape({
            id: PropTypes.string.isRequired,
            name: PropTypes.string.isRequired,
            email: PropTypes.string.isRequired
        }),
        createdTime: PropTypes.string,
        updatedTime: PropTypes.string,
        isEditable: PropTypes.bool,
        isDeletable: PropTypes.bool
    }).isRequired,
    onRemove: PropTypes.func.isRequired,
    onSave: PropTypes.func.isRequired,
    onClose: PropTypes.func,
    defaultEdit: PropTypes.bool,
    isDisabled: PropTypes.bool
};

Comment.defaultProps = {
    defaultEdit: false,
    onClose: () => {},
    isDisabled: false
};

export default Comment;
