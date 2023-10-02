import React, { ReactElement } from 'react';
import { Form, SelectOption, TextArea } from '@patternfly/react-core';

import FormLabelGroup from 'Components/PatternFly/FormLabelGroup';
import SelectSingle from 'Components/SelectSingle';

function InviteUsersForm({ formik, providers, roles, onChange }): ReactElement | null {
    const { values, touched, errors, handleBlur } = formik;

    return (
        <Form>
            <FormLabelGroup
                isRequired
                label="Emails to invite"
                fieldId="emails"
                touched={touched}
                errors={errors}
                helperText="Multiple emails should be separated with commas."
            >
                <TextArea
                    isRequired
                    type="text"
                    id="emails"
                    value={values.emails}
                    onChange={onChange}
                    onBlur={handleBlur}
                />
            </FormLabelGroup>
            <FormLabelGroup
                isRequired
                label="Provider"
                fieldId="provider"
                touched={touched}
                errors={errors}
            >
                <SelectSingle
                    id="provider"
                    value={values.provider}
                    handleSelect={onChange}
                    direction="up"
                    placeholderText="Select an auth provider"
                >
                    {providers.map(({ name }) => (
                        <SelectOption key={name} value={name} />
                    ))}
                </SelectSingle>
            </FormLabelGroup>
            <FormLabelGroup
                isRequired
                label="Role"
                fieldId="role"
                touched={touched}
                errors={errors}
            >
                <SelectSingle
                    id="role"
                    value={values.role}
                    handleSelect={onChange}
                    direction="up"
                    placeholderText="Select a role"
                >
                    {roles.map(({ name }) => (
                        <SelectOption key={name} value={name} />
                    ))}
                </SelectSingle>
            </FormLabelGroup>
        </Form>
    );
}

export default InviteUsersForm;
