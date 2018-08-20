import React from 'react';
import { reduxForm } from 'redux-form';
import { connect } from 'react-redux';
import PropTypes from 'prop-types';
import { createStructuredSelector } from 'reselect';

import { selectors } from 'reducers';

import { apiTokenFormId } from 'reducers/apitokens';
import FormField from 'Components/FormField';
import ReduxTextField from 'Components/forms/ReduxTextField';
import ReduxSelectField from 'Components/forms/ReduxSelectField';

const Fields = ({ roleOptions }) => (
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

Fields.propTypes = {
    roleOptions: PropTypes.arrayOf(
        PropTypes.shape({
            label: PropTypes.string.isRequired,
            value: PropTypes.string.isRequired
        })
    ).isRequired
};

const APITokenForm = ({ roles }) => {
    const roleOptions = roles.map(({ name }) => ({ label: name, value: name }));

    return (
        <form className="p-4 w-full mb-8" data-test-id="api-token-form">
            <Fields roleOptions={roleOptions} />
        </form>
    );
};

APITokenForm.propTypes = {
    roles: PropTypes.arrayOf(
        PropTypes.shape({
            name: PropTypes.string.isRequired
        })
    ).isRequired
};

const mapStateToProps = createStructuredSelector({
    roles: selectors.getRoles
});

const ConnectedForm = connect(mapStateToProps)(reduxForm({ form: apiTokenFormId })(APITokenForm));

export default ConnectedForm;
