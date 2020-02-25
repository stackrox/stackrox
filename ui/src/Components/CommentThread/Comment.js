import React, { useState } from 'react';
import { useForm } from 'react-hook-form';
import PropTypes from 'prop-types';
import { format } from 'date-fns';
import { Edit, Trash2, XCircle } from 'react-feather';

import dateTimeFormat from 'constants/dateTimeFormat';

import TextArea from 'Components/forms/TextArea';
import CustomDialogue from 'Components/CustomDialogue';

const regexURL = /(https?: \/\/[a-zA-Z0-9][a-zA-Z0-9-]+[a-zA-Z0-9]\.[^\s]{2,}|[a-zA-Z0-9][a-zA-Z0-9-]+[a-zA-Z0-9]\.[^\s]{2,}|https?:\/\/[a-zA-Z0-9]+\.[^\s]{2,}|[a-zA-Z0-9]+\.[^\s]{2,})/g;

const ActionButtons = ({ isEditing, canModify, onEdit, onDelete, onClose }) => {
    if (isEditing) {
        return (
            <div>
                <XCircle
                    className="h-4 w-4 ml-2 text-success-800 cursor-pointer hover:text-success-500"
                    onClick={onClose}
                />
            </div>
        );
    }
    return (
        <div className={`flex ${!canModify && 'invisible'}`}>
            <Edit
                className="h-4 w-4 mx-2 text-primary-800 cursor-pointer hover:text-primary-500"
                onClick={onEdit}
            />
            <Trash2
                className="h-4 w-4 text-primary-800 cursor-pointer hover:text-primary-500"
                onClick={onDelete}
            />
        </div>
    );
};

const Message = ({ message }) => {
    // split the message by URLs
    return message.split(regexURL).map(str => {
        // create links for each URL string
        if (str.match(regexURL)) {
            return (
                // https://mathiasbynens.github.io/rel-noopener/ explains why we add the rel="noopener noreferrer" attribute
                <a
                    href={str}
                    target="_blank"
                    rel="noopener noreferrer"
                    key={str}
                    className="text-primary-700"
                >
                    {str}
                </a>
            );
        }
        return str;
    });
};

const InputForm = ({ value, onSubmit }) => {
    const { register, handleSubmit, errors } = useForm();
    return (
        <form onSubmit={handleSubmit(onSubmit)}>
            <TextArea
                name="message"
                required
                register={register}
                errors={errors}
                rows="5"
                cols="33"
                defaultValue={value}
                placeholder="Write a comment here..."
            />
            <div className="flex justify-end">
                <input
                    className="bg-success-300 border border-success-800 p-1 rounded-sm text-sm text-success-900 uppercase hover:bg-success-400 cursor-pointer"
                    type="submit"
                    value="Save"
                />
            </div>
        </form>
    );
};

const Comment = ({ comment, onDelete, onSave, onClose, defaultEdit }) => {
    const [isEditing, setEdit] = useState(defaultEdit);
    const [isDialogueOpen, setIsDialogueOpen] = useState(false);

    const { user, createdTime, updatedTime, message, canModify } = comment;

    const isCommentUpdated = updatedTime && createdTime !== updatedTime;

    function onEdit() {
        setEdit(true);
    }

    function onCloseHandler() {
        setEdit(false);
        onClose();
    }

    function onSubmit(data) {
        onCloseHandler();
        onSave(comment, data.message);
    }

    function onDeleteHandler() {
        setIsDialogueOpen(true);
    }

    function cancelDeletion() {
        setIsDialogueOpen(false);
    }

    function confirmDeletion() {
        onDelete(comment);
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
                <div className="text-primary-800 flex flex-1">{user}</div>
                <ActionButtons
                    isEditing={isEditing}
                    canModify={canModify}
                    onEdit={onEdit}
                    onDelete={onDeleteHandler}
                    onClose={onCloseHandler}
                />
            </div>
            <div className="text-base-500 text-xs mt-1">
                {createdTime && format(createdTime, dateTimeFormat)}{' '}
                {isCommentUpdated && '(edited)'}
            </div>
            <div className="mt-2 text-primary-800 leading-normal">
                {isEditing ? (
                    <InputForm value={message} onSubmit={onSubmit} />
                ) : (
                    <Message message={message} />
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
        user: PropTypes.string,
        createdTime: PropTypes.string,
        updatedTime: PropTypes.string,
        canModify: PropTypes.bool
    }).isRequired,
    onDelete: PropTypes.func.isRequired,
    onSave: PropTypes.func.isRequired,
    onClose: PropTypes.func,
    defaultEdit: PropTypes.bool
};

Comment.defaultProps = {
    defaultEdit: false,
    onClose: () => {}
};

export default Comment;
