import React from 'react';
import { FormikProvider, useFormik } from 'formik';
import {
    PageSection,
    Title,
    Flex,
    TextInput,
    Button,
    Form,
    FormGroup,
    ActionGroup,
    FormHelperText,
    HelperText,
    HelperTextItem,
} from '@patternfly/react-core';

import { PolicyCategory } from 'types/policy.proto';
import { renamePolicyCategory } from 'services/PolicyCategoriesService';

type PolicyCategorySidePanelProps = {
    selectedCategory: PolicyCategory;
    setSelectedCategory: (selectedCategory?: PolicyCategory) => void;
    addToast: (toast) => void;
    refreshPolicyCategories: () => void;
    openDeleteModal: () => void;
};

function PolicyCategorySidePanel({
    selectedCategory,
    setSelectedCategory,
    addToast,
    refreshPolicyCategories,
    openDeleteModal,
}: PolicyCategorySidePanelProps) {
    const formik = useFormik({
        initialValues: selectedCategory,
        onSubmit: (values, { setSubmitting }) => {
            setSubmitting(false);
            const { id, name } = values;
            renamePolicyCategory(id, name)
                .then((response) => {
                    setSelectedCategory(response);
                    refreshPolicyCategories();
                })
                .catch((error) => {
                    addToast(error.message);
                })
                .finally(() => {
                    setSubmitting(false);
                });
        },
        enableReinitialize: true,
    });

    const { values, handleChange, dirty, handleSubmit } = formik;

    function onChange(event: React.FormEvent) {
        handleChange(event);
    }

    function clearSelectedCategory() {
        setSelectedCategory();
    }

    const { name } = selectedCategory;

    return (
        <>
            <PageSection isFilled variant="light" className="pf-v5-u-h-100">
                <Flex direction={{ default: 'column' }} className="pf-v5-u-h-100">
                    <Flex
                        justifyContent={{ default: 'justifyContentSpaceBetween' }}
                        fullWidth={{ default: 'fullWidth' }}
                        flexWrap={{ default: 'nowrap' }}
                    >
                        <Title headingLevel="h3">{name}</Title>
                        <Button variant="secondary" isDanger onClick={openDeleteModal}>
                            Delete category
                        </Button>
                    </Flex>
                    <FormikProvider value={formik}>
                        <Form onSubmit={handleSubmit}>
                            <FormGroup
                                fieldId="policy-category-name"
                                label="Category name"
                                isRequired
                            >
                                <TextInput
                                    id="name"
                                    type="text"
                                    value={values.name}
                                    onChange={onChange}
                                />
                                <FormHelperText>
                                    <HelperText>
                                        <HelperTextItem>
                                            Provide a descriptive and unique category name.
                                        </HelperTextItem>
                                    </HelperText>
                                </FormHelperText>
                            </FormGroup>
                            <ActionGroup>
                                {dirty && (
                                    <Button variant="primary" onClick={() => handleSubmit()}>
                                        Save
                                    </Button>
                                )}
                                <Button variant="secondary" onClick={clearSelectedCategory}>
                                    {dirty ? 'Cancel' : 'Close'}
                                </Button>
                            </ActionGroup>
                        </Form>
                    </FormikProvider>
                </Flex>
            </PageSection>
        </>
    );
}

export default PolicyCategorySidePanel;
