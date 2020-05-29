import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { reduxForm, getFormValues } from 'redux-form';

import FormFieldRequired from 'Components/forms/FormFieldRequired';
import ReduxTextField from 'Components/forms/ReduxTextField';
import PermissionsMatrix from 'Containers/AccessControl/Roles/Permissions/PermissionsMatrix/PermissionsMatrix';
import { defaultMinimalReadAccessResources } from 'constants/accessControl';

const Form = ({ handleSubmit, initialValues, onSubmit, values }) => {
    const disableNameField = !!initialValues && !!initialValues.name;
    return (
        <form
            className="w-full justify-between overflow-auto p-4"
            onSubmit={handleSubmit(onSubmit)}
            initialValues={initialValues}
        >
            <div className="mb-4 flex flex-wrap md:flex-no-wrap items-center">
                <div className="flex-shrink-0 w-full md:w-1/3 pr-8 mb-4 md:mb-0">
                    <div className="py-2 text-base-600 font-700 text-lg">
                        Role Name <FormFieldRequired empty={!values.name.trim()} />
                    </div>
                    <div data-testid="role-name" className="pb-2">
                        <ReduxTextField name="name" disabled={disableNameField} />
                    </div>
                </div>
                <div className="bg-warning-200 border-2 border-warning-400 p-3 rounded leading-normal text-warning-800">
                    <p className="mb-3">
                        <strong>Note: </strong> Users may experience issues loading certain pages
                        unless they are granted a minimum set of read permissions. If this role is
                        configured for a user, please assign at least the following read
                        permissions:{' '}
                    </p>
                    <strong>{defaultMinimalReadAccessResources.join(', ')}</strong>
                </div>
            </div>
            <div>
                <PermissionsMatrix
                    name="resourceToAccess"
                    resourceToAccess={initialValues.resourceToAccess}
                    isEditing
                />
            </div>
        </form>
    );
};

Form.propTypes = {
    handleSubmit: PropTypes.func.isRequired,
    onSubmit: PropTypes.func.isRequired,
    initialValues: PropTypes.shape({
        name: PropTypes.string,
        resourceToAccess: PropTypes.shape({}),
    }),
    values: PropTypes.shape({
        name: PropTypes.string,
    }).isRequired,
};

Form.defaultProps = {
    initialValues: null,
};

const validate = (values) => {
    const errors = {};

    if (!values.name.trim()) {
        errors.name = 'This field is required';
    }

    return errors;
};

const form = 'role-form';

const mapStateToProps = (state) => ({
    values: getFormValues(form)(state),
});

export default reduxForm({
    form,
    enableReinitialize: true,
    validate,
})(connect(mapStateToProps)(Form));
