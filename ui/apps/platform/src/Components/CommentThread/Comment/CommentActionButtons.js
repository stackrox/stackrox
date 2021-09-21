import React from 'react';
import PropTypes from 'prop-types';
import { Button, Flex } from '@patternfly/react-core';
import { PencilAltIcon, TrashIcon, TimesCircleIcon } from '@patternfly/react-icons';

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
                variant="plain"
                isDisabled={isDisabled}
                data-testid="cancel-comment-editing-button"
            >
                <TimesCircleIcon />
            </Button>
        );
    }
    return (
        <Flex spaceItems={{ default: 'spaceItemsNone' }}>
            {isEditable && (
                <Button
                    onClick={onEdit}
                    variant="plain"
                    isDisabled={isDisabled}
                    data-testid="edit-comment-button"
                >
                    <PencilAltIcon />
                </Button>
            )}
            {isDeletable && (
                <Button
                    onClick={onRemove}
                    variant="plain"
                    isDisabled={isDisabled}
                    data-testid="delete-comment-button"
                >
                    <TrashIcon />
                </Button>
            )}
        </Flex>
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
