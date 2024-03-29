import React, { ReactElement } from 'react';
import {
    Flex,
    TextInput,
    FormGroup,
    Radio,
    TextArea,
    Form,
    FormHelperText,
    HelperText,
    HelperTextItem,
} from '@patternfly/react-core';
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
                    <FormGroup fieldId="policy-name" label="Name" isRequired>
                        <FormHelperText>
                            <HelperText>
                                <HelperTextItem>
                                    Provide a descriptive and unique policy name
                                </HelperTextItem>
                            </HelperText>
                        </FormHelperText>
                        <TextInput
                            id={field.name}
                            name={field.name}
                            value={field.value}
                            onChange={(event, _value) => onChange(_value, event)}
                            isRequired
                        />
                    </FormGroup>
                )}
            </Field>
            <FormGroup fieldId="policy-severity" label="Severity" isRequired>
                <FormHelperText>
                    <HelperText>
                        <HelperTextItem>Select a severity level for this policy</HelperTextItem>
                    </HelperText>
                </FormHelperText>
                <Flex direction={{ default: 'row' }}>
                    <Field name="severity" type="radio" value="LOW_SEVERITY">
                        {({ field }) => (
                            <Radio
                                name={field.name}
                                value={field.value}
                                onChange={(event, _value) => onChange(_value, event)}
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
                                onChange={(event, _value) => onChange(_value, event)}
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
                                onChange={(event, _value) => onChange(_value, event)}
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
                                onChange={(event, _value) => onChange(_value, event)}
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
                    <FormGroup fieldId="policy-description" label="Description">
                        <TextArea
                            id={field.name}
                            name={field.name}
                            value={field.value}
                            onChange={(event, _value) => onChange(_value, event)}
                        />
                        <FormHelperText>
                            <HelperText>
                                <HelperTextItem>Enter details about the policy</HelperTextItem>
                            </HelperText>
                        </FormHelperText>
                    </FormGroup>
                )}
            </Field>
            <Field name="rationale">
                {({ field }) => (
                    <FormGroup fieldId="policy-rationale" label="Rationale">
                        <TextArea
                            id={field.name}
                            name={field.name}
                            value={field.value}
                            onChange={(event, _value) => onChange(_value, event)}
                        />
                        <FormHelperText>
                            <HelperText>
                                <HelperTextItem>
                                    Enter an explanation about why this policy exists
                                </HelperTextItem>
                            </HelperText>
                        </FormHelperText>
                    </FormGroup>
                )}
            </Field>
            <Field name="remediation">
                {({ field }) => (
                    <FormGroup fieldId="policy-guidance" label="Guidance">
                        <TextArea
                            id={field.name}
                            name={field.name}
                            value={field.value}
                            onChange={(event, _value) => onChange(_value, event)}
                        />
                        <FormHelperText>
                            <HelperText>
                                <HelperTextItem>
                                    Enter steps to resolve the violations of this policy
                                </HelperTextItem>
                            </HelperText>
                        </FormHelperText>
                    </FormGroup>
                )}
            </Field>
        </Form>
    );
}

export default PolicyMetadataFormSection;
