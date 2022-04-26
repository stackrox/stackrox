import React from 'react';
import PropTypes from 'prop-types';
import { FormGroup, Radio, TextInput } from '@patternfly/react-core';
import { Field } from 'formik';

import { POLICY_DUPE_ACTIONS } from './PolicyImport.utils';

const RenamePolicySection = ({ changeRadio, changeText }) => {
    return (
        <div>
            <Field name="resolution">
                {({ field }) => (
                    <Radio
                        name={field.name}
                        value="overwrite"
                        label="Rename incoming policy"
                        id="policy-rename-radio"
                        isChecked={field.value === POLICY_DUPE_ACTIONS.RENAME}
                        onChange={changeRadio(
                            field.onChange,
                            field.name,
                            POLICY_DUPE_ACTIONS.RENAME
                        )}
                    />
                )}
            </Field>
            <Field name="newName">
                {({ field, form }) => {
                    const isDisabled = form.values.resolution !== POLICY_DUPE_ACTIONS.RENAME;
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

RenamePolicySection.propTypes = {
    changeRadio: PropTypes.func.isRequired,
    changeText: PropTypes.func.isRequired,
};

export default RenamePolicySection;
