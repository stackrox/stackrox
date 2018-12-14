import React from 'react';
import PropTypes from 'prop-types';
import { reduxForm } from 'redux-form';

import ReduxTextField from 'Components/forms/ReduxTextField';
import PermissionsMatrix from 'Containers/AccessControl/Roles/Permissions/PermissionsMatrix/PermissionsMatrix';
import { defaultPermissions } from 'constants/accessControl';

const Form = props => {
    const { handleSubmit, initialValues, onSubmit } = props;
    const resourceToAccess = initialValues.resourceToAccess || defaultPermissions;
    const disableNameField = !!initialValues && !!initialValues.name;
    return (
        <form
            className="w-full justify-between overflow-auto"
            onSubmit={handleSubmit(onSubmit)}
            initialvalues={initialValues}
        >
            <div className="mb-4">
                <div className="py-2 text-base-600 font-700">Role Name</div>
                <div className="w-1/3" data-test-id="role-name">
                    <ReduxTextField name="name" disabled={disableNameField} />
                </div>
            </div>
            <div>
                <div className="py-2 text-base-600 font-700">Permissions</div>
                <PermissionsMatrix
                    name="resourceToAccess"
                    resourceToAccess={resourceToAccess}
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
        resourceToAccess: PropTypes.shape({})
    })
};

Form.defaultProps = {
    initialValues: null
};

export default reduxForm({
    form: 'role-form'
})(Form);
