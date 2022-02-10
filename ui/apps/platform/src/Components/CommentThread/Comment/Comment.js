import React, { useState } from 'react';
import PropTypes from 'prop-types';
import { format } from 'date-fns';
import {
    Card,
    CardHeader,
    CardBody,
    CardActions,
    CardTitle,
    Flex,
    FlexItem,
    Modal,
    ModalVariant,
    Button,
    ButtonVariant,
} from '@patternfly/react-core';

import dateTimeFormat from 'constants/dateTimeFormat';
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
        <>
            <Card
                className={`${
                    isEditing ? 'pf-u-background-color-success' : 'pf-u-background-color-info'
                }`}
                isFlat
            >
                <CardHeader>
                    <CardTitle data-testid="comment-header">
                        <Flex direction={{ default: 'column' }}>
                            <FlexItem data-testid="comment-header-title" className="pf-u-mb-xs">
                                {textHeader}
                            </FlexItem>
                            <FlexItem
                                className="pf-u-font-size-sm"
                                data-testid="comment-header-subtitle"
                            >
                                {createdTime && format(createdTime, dateTimeFormat)}
                                {isCommentUpdated && ' (edited)'}
                            </FlexItem>
                        </Flex>
                    </CardTitle>
                    <CardActions>
                        <CommentActionButtons
                            isEditing={isEditing}
                            isEditable={isEditable}
                            isDeletable={isDeletable}
                            onEdit={onEdit}
                            onRemove={onRemoveHandler}
                            onClose={onCloseHandler}
                            isDisabled={isDisabled}
                        />
                    </CardActions>
                </CardHeader>
                <CardBody data-testid="comment-message">
                    {isEditing ? (
                        <CommentForm initialFormValues={initialFormValues} onSubmit={onSubmit} />
                    ) : (
                        <CommentMessage message={message} />
                    )}
                </CardBody>
            </Card>
            <Modal
                variant={ModalVariant.small}
                isOpen={isDialogueOpen}
                actions={[
                    <Button key="confirm" variant={ButtonVariant.danger} onClick={confirmDeletion}>
                        Delete
                    </Button>,
                    <Button key="cancel" variant="link" onClick={cancelDeletion}>
                        Cancel
                    </Button>,
                ]}
                onClose={cancelDeletion}
                aria-label="Delete comment confirmation"
            >
                Delete Comment?
            </Modal>
        </>
    );
};

Comment.propTypes = {
    comment: PropTypes.shape({
        id: PropTypes.string,
        message: PropTypes.string,
        user: PropTypes.shape({
            id: PropTypes.string.isRequired,
            name: PropTypes.string.isRequired,
            email: PropTypes.string.isRequired,
        }),
        createdTime: PropTypes.string,
        updatedTime: PropTypes.string,
        isEditable: PropTypes.bool,
        isDeletable: PropTypes.bool,
    }).isRequired,
    onRemove: PropTypes.func.isRequired,
    onSave: PropTypes.func.isRequired,
    onClose: PropTypes.func,
    defaultEdit: PropTypes.bool,
    isDisabled: PropTypes.bool,
};

Comment.defaultProps = {
    defaultEdit: false,
    onClose: () => {},
    isDisabled: false,
};

export default Comment;
