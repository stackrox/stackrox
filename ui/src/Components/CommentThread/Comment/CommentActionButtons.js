import React from 'react';
import PropTypes from 'prop-types';
import { Edit, Trash2, XCircle } from 'react-feather';

import Button from 'Components/Button';

const CommentActionButtons = ({
    isEditing,
    isModifiable,
    onEdit,
    onRemove,
    onClose,
    isDisabled
}) => {
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

CommentActionButtons.propTypes = {
    isEditing: PropTypes.bool,
    isModifiable: PropTypes.bool,
    onEdit: PropTypes.func.isRequired,
    onRemove: PropTypes.func.isRequired,
    onClose: PropTypes.func.isRequired,
    isDisabled: PropTypes.bool
};

CommentActionButtons.defaultProps = {
    isEditing: false,
    isDisabled: false,
    isModifiable: false
};

export default CommentActionButtons;
