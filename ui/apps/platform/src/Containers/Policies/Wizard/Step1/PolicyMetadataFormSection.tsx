import React, { ReactElement } from 'react';
import { Flex, TextInput, Radio, TextArea, Form } from '@patternfly/react-core';
import { FormikContextType, useFormikContext } from 'formik';

import FormLabelGroup from 'Components/PatternFly/FormLabelGroup';
import { ClientPolicy } from 'types/policy.proto';

import PolicyCategoriesSelectField from './PolicyCategoriesSelectField';

function PolicyMetadataFormSection(): ReactElement {
    const {
        errors,
        handleChange,
        handleBlur,
        setFieldValue,
        touched,
        values,
    }: FormikContextType<ClientPolicy> = useFormikContext();

    function handleSeverityChange(severity: string) {
        setFieldValue('severity', severity);
    }

    return (
        <Form>
            <FormLabelGroup
                isRequired
                label="Name"
                fieldId="name"
                errors={errors}
                touched={touched}
                helperText={'Provide a descriptive and unique policy name'}
            >
                <TextInput
                    isRequired
                    type="text"
                    id="name"
                    name="name"
                    value={values.name}
                    validated={errors?.name && touched?.name ? 'error' : 'default'}
                    onChange={handleChange}
                    onBlur={handleBlur}
                />
            </FormLabelGroup>
            <FormLabelGroup
                isRequired
                label="Severity"
                fieldId="severity"
                errors={errors}
                touched={touched}
                helperText={'Select a severity level for this policy'}
            >
                <Flex direction={{ default: 'row' }}>
                    <Radio
                        name="severity"
                        value="LOW_SEVERITY"
                        onChange={() => handleSeverityChange('LOW_SEVERITY')}
                        label="Low"
                        id="policy-severity-radio-low"
                        isChecked={values.severity === 'LOW_SEVERITY'}
                        onBlur={handleBlur}
                    />
                    <Radio
                        name="severity"
                        value="MEDIUM_SEVERITY"
                        onChange={() => handleSeverityChange('MEDIUM_SEVERITY')}
                        label="Medium"
                        id="policy-severity-radio-medium"
                        isChecked={values.severity === 'MEDIUM_SEVERITY'}
                        onBlur={handleBlur}
                    />
                    <Radio
                        name="severity"
                        value="HIGH_SEVERITY"
                        onChange={() => handleSeverityChange('HIGH_SEVERITY')}
                        label="High"
                        id="policy-severity-radio-high"
                        isChecked={values.severity === 'HIGH_SEVERITY'}
                        onBlur={handleBlur}
                    />
                    <Radio
                        name="severity"
                        value="CRITICAL_SEVERITY"
                        onChange={() => handleSeverityChange('CRITICAL_SEVERITY')}
                        label="Critical"
                        id="policy-severity-radio-critical"
                        isChecked={values.severity === 'CRITICAL_SEVERITY'}
                        onBlur={handleBlur}
                    />
                </Flex>
            </FormLabelGroup>
            <PolicyCategoriesSelectField />
            <FormLabelGroup
                label="Description"
                fieldId="description"
                errors={errors}
                touched={touched}
                helperText={'Enter a description of the policy'}
            >
                <TextArea
                    id="description"
                    name="description"
                    value={values.description}
                    onChange={handleChange}
                    onBlur={handleBlur}
                />
            </FormLabelGroup>
            <FormLabelGroup
                label="Rationale"
                fieldId="rationale"
                errors={errors}
                touched={touched}
                helperText={'Enter an explanation about why this policy exists'}
            >
                <TextArea
                    id="rationale"
                    name="rationale"
                    value={values.rationale}
                    onChange={handleChange}
                    onBlur={handleBlur}
                />
            </FormLabelGroup>
            <FormLabelGroup
                label="Guidance"
                fieldId="remediation"
                errors={errors}
                touched={touched}
                helperText={'Enter steps to resolve the violations of this policy'}
            >
                <TextArea
                    id="remediation"
                    name="remediation"
                    value={values.remediation}
                    onChange={handleChange}
                    onBlur={handleBlur}
                />
            </FormLabelGroup>
        </Form>
    );
}

export default PolicyMetadataFormSection;
