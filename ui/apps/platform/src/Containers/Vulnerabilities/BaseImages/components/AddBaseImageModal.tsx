import React, { useState } from 'react';
import {
    Modal,
    Form,
    FormGroup,
    TextInput,
    Button,
    FormHelperText,
    HelperText,
    HelperTextItem,
} from '@patternfly/react-core';

type AddBaseImageModalProps = {
    isOpen: boolean;
    onClose: () => void;
    onAdd: (name: string) => void;
};

function AddBaseImageModal({ isOpen, onClose, onAdd }: AddBaseImageModalProps) {
    const [imageName, setImageName] = useState('');
    const [validationError, setValidationError] = useState<string | null>(null);

    const handleAdd = () => {
        // Validate input
        if (!imageName.trim()) {
            setValidationError('Base image name is required');
            return;
        }

        if (!imageName.includes(':')) {
            setValidationError('Image name must include a tag (e.g., ubuntu:22.04)');
            return;
        }

        // Clear validation and add image
        setValidationError(null);
        onAdd(imageName);
        setImageName('');
        onClose();
    };

    const handleClose = () => {
        setImageName('');
        setValidationError(null);
        onClose();
    };

    return (
        <Modal
            variant="small"
            title="Add base image"
            isOpen={isOpen}
            onClose={handleClose}
            actions={[
                <Button key="add" variant="primary" onClick={handleAdd}>
                    Add
                </Button>,
                <Button key="cancel" variant="link" onClick={handleClose}>
                    Cancel
                </Button>,
            ]}
        >
            <Form>
                <FormGroup label="Base image name" isRequired fieldId="base-image-name">
                    <TextInput
                        id="base-image-name"
                        type="text"
                        value={imageName}
                        onChange={(_event, value) => {
                            setImageName(value);
                            if (validationError) {
                                setValidationError(null);
                            }
                        }}
                        placeholder="e.g., ubuntu:22.04, alpine:3.18"
                        validated={validationError ? 'error' : 'default'}
                    />
                    {validationError ? (
                        <FormHelperText>
                            <HelperText>
                                <HelperTextItem variant="error">{validationError}</HelperTextItem>
                            </HelperText>
                        </FormHelperText>
                    ) : (
                        <FormHelperText>
                            <HelperText>
                                <HelperTextItem>Enter the base image name with tag</HelperTextItem>
                            </HelperText>
                        </FormHelperText>
                    )}
                </FormGroup>
            </Form>
        </Modal>
    );
}

export default AddBaseImageModal;
