import React from 'react';
import { reduxForm } from 'redux-form';

import { apiTokenFormId } from 'reducers/apitokens';
import FormField from 'Components/FormField';
import ReduxTextField from 'Components/forms/ReduxTextField';
import ReduxSelectField from 'Components/forms/ReduxSelectField';

// TODO(viswa): Hard coding these in both the UI and the backend is temporary.
// Fix the backend to return these roles in an API.
const roleOptions = [
    { label: 'Sensor Creator', value: 'Sensor Creator' },
    { label: 'Admin', value: 'Admin' }
];

const Fields = () => (
    <React.Fragment>
        <FormField label="Token Name" required>
            <ReduxTextField name="name" />
        </FormField>
        <FormField label="Role" required>
            <ReduxSelectField
                name="role"
                placeholder="The role you want this token to have"
                options={roleOptions}
            />
        </FormField>
    </React.Fragment>
);

const APITokenForm = () => (
    <form className="p-4 w-full mb-8" data-test-id="api-token-form">
        <Fields />
    </form>
);

const ConnectedForm = reduxForm({ form: apiTokenFormId })(APITokenForm);

export default ConnectedForm;
