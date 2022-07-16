import React from 'react';
import { FormikProvider, useFormik } from 'formik';
import {
    PageSection,
    Title,
    Flex,
    FlexItem,
    TextInput,
    Button,
    Form,
    FormGroup,
    ActionGroup,
} from '@patternfly/react-core';

import { PolicyCategory } from 'types/policy.proto';
import { renamePolicyCategory } from 'services/PolicyCategoriesService';

type PolicyCategoriesSidePanelProps = {
    selectedCategory: PolicyCategory;
    setSelectedCategory: (selectedCategory?: PolicyCategory) => void;
};

function PolicyCategorySidePanel({
    selectedCategory,
    setSelectedCategory,
}: PolicyCategoriesSidePanelProps) {
    const formik = useFormik({
        initialValues: selectedCategory,
        onSubmit: (values, { setSubmitting }) => {
            setSubmitting(false);
            const { id, name } = values;
            renamePolicyCategory(id, name)
                .then(() => {
                    setSelectedCategory(values);
                })
                .catch((error) => {
                    console.error(error);
                })
                .finally(() => {
                    setSubmitting(false);
                });
        },
    });

    return (
        <>
            <PageSection isFilled variant="light">
                <Flex direction={{ default: 'column' }}>
                    <Flex
                        justifyContent={{ default: 'justifyContentSpaceBetween' }}
                        fullWidth={{ default: 'fullWidth' }}
                    >
                        <Title headingLevel="h3">{selectedCategory}</Title>
                        <Button variant="secondary" isDanger>
                            Delete category
                        </Button>
                    </Flex>
                    <FlexItem>
                        <FormikProvider value={formik}>
                            <Form>
                                <FormGroup
                                    fieldId="policy-category-name"
                                    label="Category name"
                                    isRequired
                                    helperText="Provide a descriptive and unique category name."
                                >
                                    <TextInput
                                        id="policy-category-name"
                                        type="text"
                                        value={selectedCategory.name}
                                        onChange={() => {}}
                                    />
                                </FormGroup>
                                <ActionGroup>
                                    <Button variant="primary">Save</Button>
                                    <Button
                                        variant="secondary"
                                        onClick={() => setSelectedCategory()}
                                    >
                                        Cancel
                                    </Button>
                                </ActionGroup>
                            </Form>
                        </FormikProvider>
                    </FlexItem>
                </Flex>
            </PageSection>
        </>
    );
}

export default PolicyCategorySidePanel;
