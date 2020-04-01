import React, { useState } from 'react';
import { Formik } from 'formik';
import PropTypes from 'prop-types';
import { format } from 'date-fns';
import { Edit, Trash2, XCircle } from 'react-feather';
import { object, string } from 'yup';

import { httpURLPattern, isValidURL } from 'utils/urlUtils';
import dateTimeFormat from 'constants/dateTimeFormat';
import CustomDialogue from 'Components/CustomDialogue';
import Button from 'Components/Button';

const ActionButtons = ({ isEditing, isModifiable, onEdit, onRemove, onClose, isDisabled }) => {
    if (isEditing) {
        return (
            <Button
                onClick={onClose}
                icon={
                    <XCircle className="h-4 w-4 ml-2 text-success-800 cursor-pointer hover:text-success-500" />
                }
                disabled={isDisabled}
            />
        );
    }
    return (
        <div className={`flex ${!isModifiable && 'invisible'}`}>
            <Button
                onClick={onEdit}
                icon={
                    <Edit className="h-4 w-4 mx-2 text-primary-800 cursor-pointer hover:text-primary-500" />
                }
                disabled={isDisabled}
            />
            <Button
                onClick={onRemove}
                icon={
                    <Trash2 className="h-4 w-4 text-primary-800 cursor-pointer hover:text-primary-500" />
                }
                disabled={isDisabled}
            />
        </div>
    );
};

const Message = ({ message }) => {
    // split the message by URLs
    return message.split(httpURLPattern).map(str => {
        // create links for each URL string
        if (isValidURL(str)) {
            return (
                // https://mathiasbynens.github.io/rel-noopener/ explains why we add the rel="noopener noreferrer" attribute
                <a
                    href={str}
                    target="_blank"
                    rel="noopener noreferrer"
                    key={str}
                    className="text-primary-700"
                    data-testid="comment-link"
                >
                    {str}
                </a>
            );
        }
        return str;
    });
};

const CommentForm = ({ initialFormValues, onSubmit }) => {
    return (
        <Formik
            initialValues={initialFormValues}
            validationSchema={object().shape({
                message: string()
                    .trim()
                    .required('This field is required')
            })}
            onSubmit={onSubmit}
        >
            {({ values, errors, handleChange, handleBlur, handleSubmit }) => (
                <form onSubmit={handleSubmit}>
                    <textarea
                        data-testid="comment-textarea"
                        className="form-textarea text-base border border-base-400 leading-normal p-1 w-full"
                        name="message"
                        rows="5"
                        cols="33"
                        placeholder="Write a comment here..."
                        onChange={handleChange}
                        onBlur={handleBlur}
                        value={values.message}
                    />
                    {errors && errors.message && (
                        <span className="text-alert-700">{errors.message}</span>
                    )}
                    <div className="flex justify-end">
                        <button
                            className="bg-success-300 border border-success-800 p-1 rounded-sm text-sm text-success-900 uppercase hover:bg-success-400 cursor-pointer"
                            type="submit"
                        >
                            Save
                        </button>
                    </div>
                </form>
            )}
        </Formik>
    );
};

const Comment = ({ comment, onRemove, onSave, onClose, defaultEdit, isDisabled }) => {
    const [isEditing, setEdit] = useState(defaultEdit);
    const [isDialogueOpen, setIsDialogueOpen] = useState(false);

    const { id, user, createdTime, updatedTime, message, isModifiable } = comment;

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
                <ActionButtons
                    isEditing={isEditing}
                    isModifiable={isModifiable}
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
        user: PropTypes.shape({
            id: PropTypes.string,
            name: PropTypes.string,
            email: PropTypes.string
        }),
        createdTime: PropTypes.string,
        updatedTime: PropTypes.string,
        isModifiable: PropTypes.bool
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
