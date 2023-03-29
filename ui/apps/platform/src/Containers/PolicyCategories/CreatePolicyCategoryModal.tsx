import React from 'react';
import * as yup from 'yup';
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
        onSubmit: (values, { setSubmitting, resetForm }) => {
            setSubmitting(false);
            postPolicyCategory(values)
                .then(() => {
                    setTimeout(refreshPolicyCategories, 200);
                    addToast('Successfully added category');
                })
                .catch((error) => {
                    addToast(error.message);
                })
                .finally(() => {
                    setSubmitting(false);
                    resetForm();
                    onClose();
                });
        },
        validateOnMount: true,
        validationSchema: yup.object().shape({
            name: yup
                .string()
                .min(5, 'Policy category name must be at least 5 characters long')
                .max(128, 'Policy category name must be less than 128 characters long')
                .required(),
        }),
    });

    const { values, handleChange, handleSubmit, resetForm, isValid } = formik;

    function onChange(_value, event) {
        handleChange(event);
    }

    function onCancel() {
        resetForm();
        onClose();
    }

    return (
        <Modal
            title="Create category"
            isOpen={isOpen}
            variant={ModalVariant.small}
            onClose={onCancel}
            data-testid="create-category-modal"
            aria-label="Create category"
            hasNoBodyWrapper
        >
            <ModalBoxBody>
                <FormikProvider value={formik}>
                    <Form onSubmit={handleSubmit}>
                        <FormGroup
                            fieldId="name"
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
                    isDisabled={!isValid}
                    type="submit"
                >
                    Create
                </Button>
                <Button key="cancel" variant="link" onClick={onCancel}>
                    Cancel
                </Button>
            </ModalBoxFooter>
        </Modal>
    );
}

export default CreatePolicyCategoryModal;
