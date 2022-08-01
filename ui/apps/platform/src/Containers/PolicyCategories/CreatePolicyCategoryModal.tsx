import React from 'react';
import {
    Modal,
    ModalVariant,
    ModalBoxBody,
    ModalBoxFooter,
    Button,
    Form,
    FormGroup,
    TextInput,
} from '@patternfly/react-core';
import { FormikProvider, useFormik } from 'formik';

import { PolicyCategory } from 'types/policy.proto';
import { postPolicyCategory } from 'services/PolicyCategoriesService';

type CreatePolicyCategoryModalType = {
    isOpen: boolean;
    onClose: () => void;
    addToast: (toast) => void;
    refreshPolicyCategories: () => void;
};

const emptyPolicyCategory = {
    id: '',
    name: '',
    isDefault: false,
};

function CreatePolicyCategoryModal({
    isOpen,
    onClose,
    addToast,
    refreshPolicyCategories,
}: CreatePolicyCategoryModalType) {
    const formik = useFormik({
        initialValues: emptyPolicyCategory as PolicyCategory,
        onSubmit: (values, { setSubmitting }) => {
            setSubmitting(false);
            postPolicyCategory(values)
                .then(() => {
                    setTimeout(refreshPolicyCategories, 200);
                })
                .catch((error) => {
                    addToast(error.message);
                })
                .finally(() => {
                    setSubmitting(false);
                    onClose();
                });
        },
    });

    const { values, handleChange, handleSubmit } = formik;

    function onChange(_value, event) {
        handleChange(event);
    }

    return (
        <Modal
            title="Create category"
            isOpen={isOpen}
            variant={ModalVariant.small}
            onClose={onClose}
            data-testid="create-category-modal"
            aria-label="Create category"
            hasNoBodyWrapper
        >
            <ModalBoxBody>
                <FormikProvider value={formik}>
                    <Form>
                        <FormGroup
                            fieldId="policy-category-name"
                            label="Category name"
                            isRequired
                            helperText="Provide a descriptive and unique category name."
                        >
                            <TextInput
                                id="name"
                                type="text"
                                value={values.name}
                                onChange={onChange}
                            />
                        </FormGroup>
                    </Form>
                </FormikProvider>
            </ModalBoxBody>
            <ModalBoxFooter>
                <Button
                    key="create"
                    variant="primary"
                    onClick={() => handleSubmit()}
                    // isDisabled={}
                >
                    Create
                </Button>
                <Button key="cancel" variant="link" onClick={onClose}>
                    Cancel
                </Button>
            </ModalBoxFooter>
        </Modal>
    );
}

export default CreatePolicyCategoryModal;
