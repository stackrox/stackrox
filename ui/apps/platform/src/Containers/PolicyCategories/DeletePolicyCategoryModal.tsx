import React from 'react';
import { Modal, ModalVariant, ModalBoxBody, ModalBoxFooter, Button } from '@patternfly/react-core';

import { PolicyCategory } from 'types/policy.proto';
import { deletePolicyCategory } from 'services/PolicyCategoriesService';

type DeletePolicyCategoryModalProps = {
    isOpen: boolean;
    selectedCategory: PolicyCategory;
    onClose: () => void;
};

function DeletePolicyCategoryModal({
    isOpen,
    selectedCategory,
    onClose,
}: DeletePolicyCategoryModalProps) {
    function handleCancel() {}

    function handleClose() {
        onClose();
    }

    function handleSubmit() {}

    return (
        <Modal
            title="Create category"
            isOpen={isOpen}
            variant={ModalVariant.small}
            onClose={handleClose}
            data-testid="create-category-modal"
            aria-label="Create category"
            hasNoBodyWrapper
        >
            <ModalBoxBody>delete category modal, list of policies here</ModalBoxBody>
            <ModalBoxFooter>
                <Button
                    key="delete"
                    variant="primary"
                    onClick={() => handleSubmit()}
                    // isDisabled={}
                >
                    Delete
                </Button>
                <Button key="cancel" variant="link" onClick={handleCancel}>
                    Cancel
                </Button>
            </ModalBoxFooter>
        </Modal>
    );
}

export default DeletePolicyCategoryModal;
