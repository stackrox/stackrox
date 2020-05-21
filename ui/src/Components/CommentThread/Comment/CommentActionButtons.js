import React from 'react';
import PropTypes from 'prop-types';
import { Edit, Trash2, XCircle } from 'react-feather';

import Button from 'Components/Button';

const CommentActionButtons = ({
    isEditing,
    isEditable,
    isDeletable,
    onEdit,
    onRemove,
    onClose,
    isDisabled,
}) => {
    if (isEditing) {
        return (
            <Button
                onClick={onClose}
                icon={
                    <XCircle className="h-4 w-4 ml-2 text-success-800 cursor-pointer hover:text-success-500" />
                }
                disabled={isDisabled}
                dataTestId="cancel-comment-editing-button"
            />
        );
    }
    return (
        <div className="flex">
            {isEditable && (
                <Button
                    onClick={onEdit}
                    icon={
                        <Edit className="h-4 w-4 mx-2 text-primary-800 cursor-pointer hover:text-primary-500" />
                    }
                    disabled={isDisabled}
                    dataTestId="edit-comment-button"
                />
            )}
            {isDeletable && (
                <Button
                    onClick={onRemove}
                    icon={
                        <Trash2 className="h-4 w-4 text-primary-800 cursor-pointer hover:text-primary-500" />
                    }
                    disabled={isDisabled}
                    dataTestId="delete-comment-button"
                />
            )}
        </div>
    );
};

CommentActionButtons.propTypes = {
    isEditing: PropTypes.bool,
    isEditable: PropTypes.bool,
    isDeletable: PropTypes.bool,
    onEdit: PropTypes.func.isRequired,
    onRemove: PropTypes.func.isRequired,
    onClose: PropTypes.func.isRequired,
    isDisabled: PropTypes.bool,
};

CommentActionButtons.defaultProps = {
    isEditing: false,
    isDisabled: false,
    isEditable: false,
    isDeletable: false,
};

export default CommentActionButtons;
