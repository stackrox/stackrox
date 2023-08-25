import React, { ReactElement } from 'react';
import { Flex, TextInput, FormGroup, Radio, TextArea, Form } from '@patternfly/react-core';
import { Field, useFormikContext } from 'formik';

import PolicyCategoriesSelectField from './PolicyCategoriesSelectField';

function PolicyMetadataFormSection(): ReactElement {
    const { handleChange } = useFormikContext();

    function onChange(_value, event) {
        handleChange(event);
    }
    return (
        <Form>
            <Field name="name">
                {({ field }) => (
                    <FormGroup
                        helperText="Provide a descriptive and unique policy name"
                        fieldId="policy-name"
                        label="Name"
                        isRequired
                    >
                        <TextInput
                            id={field.name}
                            name={field.name}
                            value={field.value}
                            onChange={onChange}
                            isRequired
                        />
                    </FormGroup>
                )}
            </Field>
            <FormGroup
                helperText="Select a severity level for this policy"
                fieldId="policy-severity"
                label="Severity"
                isRequired
            >
                <Flex direction={{ default: 'row' }}>
                    <Field name="severity" type="radio" value="LOW_SEVERITY">
                        {({ field }) => (
                            <Radio
                                name={field.name}
                                value={field.value}
                                onChange={onChange}
                                label="Low"
                                id="policy-severity-radio-low"
                                isChecked={field.checked}
                            />
                        )}
                    </Field>
                    <Field name="severity" type="radio" value="MEDIUM_SEVERITY">
                        {({ field }) => (
                            <Radio
                                name={field.name}
                                value={field.value}
                                onChange={onChange}
                                label="Medium"
                                id="policy-severity-radio-medium"
                                isChecked={field.checked}
                            />
                        )}
                    </Field>
                    <Field name="severity" type="radio" value="HIGH_SEVERITY">
                        {({ field }) => (
                            <Radio
                                name={field.name}
                                value={field.value}
                                onChange={onChange}
                                label="High"
                                id="policy-severity-radio-high"
                                isChecked={field.checked}
                            />
                        )}
                    </Field>
                    <Field name="severity" type="radio" value="CRITICAL_SEVERITY">
                        {({ field }) => (
                            <Radio
                                name={field.name}
                                value={field.value}
                                onChange={onChange}
                                label="Critical"
                                id="policy-severity-radio-critical"
                                isChecked={field.checked}
                            />
                        )}
                    </Field>
                </Flex>
            </FormGroup>
            <PolicyCategoriesSelectField />
            <Field name="description">
                {({ field }) => (
                    <FormGroup
                        helperText="Enter details about the policy"
                        fieldId="policy-description"
                        label="Description"
                    >
                        <TextArea
                            id={field.name}
                            name={field.name}
                            value={field.value}
                            onChange={onChange}
                        />
                    </FormGroup>
                )}
            </Field>
            <Field name="rationale">
                {({ field }) => (
                    <FormGroup
                        helperText="Enter an explanation about why this policy exists"
                        fieldId="policy-rationale"
                        label="Rationale"
                    >
                        <TextArea
                            id={field.name}
                            name={field.name}
                            value={field.value}
                            onChange={onChange}
                        />
                    </FormGroup>
                )}
            </Field>
            <Field name="remediation">
                {({ field }) => (
                    <FormGroup
                        helperText="Enter steps to resolve the violations of this policy"
                        fieldId="policy-guidance"
                        label="Guidance"
                    >
                        <TextArea
                            id={field.name}
                            name={field.name}
                            value={field.value}
                            onChange={onChange}
                        />
                    </FormGroup>
                )}
            </Field>
        </Form>
    );
}

export default PolicyMetadataFormSection;
