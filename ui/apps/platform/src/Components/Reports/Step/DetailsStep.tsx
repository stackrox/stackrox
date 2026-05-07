import type { ReactElement } from 'react';
import { Flex, Form, PageSection, TextArea, TextInput, Title } from '@patternfly/react-core';
import type { FormikProps } from 'formik';

import FormLabelGroup from 'Components/PatternFly/FormLabelGroup';

import type { DetailsType, ReportPageAction } from '../reports.types';

export type DetailsStepProps<T extends DetailsType = DetailsType> = {
    formik: FormikProps<T>;
    pageAction: ReportPageAction;
};

function DetailsStep<T extends DetailsType = DetailsType>({
    formik,
    pageAction,
}: DetailsStepProps<T>): ReactElement {
    return (
        <PageSection>
            <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsLg' }}>
                <Title headingLevel="h2">Details</Title>
                <Form>
                    <FormLabelGroup
                        label="Name"
                        isRequired
                        fieldId="name"
                        errors={formik.errors}
                        touched={formik.touched}
                    >
                        <TextInput
                            type="text"
                            id="name"
                            name="name"
                            value={formik.values.name}
                            isDisabled={pageAction === 'edit'}
                            isRequired
                            validated={
                                formik.errors?.name && formik.touched?.name ? 'error' : 'default'
                            }
                            onChange={formik.handleChange}
                            onBlur={formik.handleBlur}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        label="Description"
                        fieldId="description"
                        errors={formik.errors}
                        touched={formik.touched}
                    >
                        <TextArea
                            type="text"
                            id="description"
                            name="description"
                            value={formik.values.description}
                            onChange={formik.handleChange}
                            onBlur={formik.handleBlur}
                        />
                    </FormLabelGroup>
                </Form>
            </Flex>
        </PageSection>
    );
}

export default DetailsStep;
