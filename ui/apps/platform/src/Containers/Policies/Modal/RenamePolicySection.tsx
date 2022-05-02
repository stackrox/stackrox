import React from 'react';
import { FormGroup, Radio, TextInput } from '@patternfly/react-core';
import { Field } from 'formik';

type RenamePolicySectionProps = {
    changeRadio: (handler, name, value) => () => void;
    changeText: (handler, name) => (value) => void;
};

const RenamePolicySection = ({ changeRadio, changeText }: RenamePolicySectionProps) => {
    return (
        <div>
            <Field name="resolution">
                {({ field }) => (
                    <Radio
                        name={field.name}
                        value="overwrite"
                        label="Rename incoming policy"
                        id="policy-rename-radio"
                        isChecked={field.value === 'rename'}
                        onChange={changeRadio(field.onChange, field.name, 'rename')}
                    />
                )}
            </Field>
            <Field name="newName">
                {({ field, form }) => {
                    const isDisabled = form.values.resolution !== 'rename';
                    const validated =
                        form.touched.newName && form.errors.newName ? 'error' : 'default';
                    return (
                        <FormGroup
                            helperTextInvalid={form.errors.newName}
                            fieldId="policy-rename"
                            validated={validated}
                            className="pf-u-pt-sm"
                        >
                            <TextInput
                                name={field.name}
                                value={field.value}
                                label="Rename incoming policy"
                                id="policy-rename"
                                onChange={changeText(field.onChange, field.name)}
                                onBlur={field.onBlur}
                                isDisabled={isDisabled}
                                validated={validated}
                            />
                        </FormGroup>
                    );
                }}
            </Field>
        </div>
    );
};

export default RenamePolicySection;
