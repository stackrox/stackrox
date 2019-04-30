import React from 'react';
import PropTypes from 'prop-types';
import { Field } from 'redux-form';

import AccessField from 'Containers/AccessControl/Roles/Permissions/PermissionsMatrix/AccessField';

const TableRows = props => {
    const { resourceToAccess, name, isEditing } = props;
    if (!resourceToAccess) return null;
    return Object.keys(resourceToAccess)
        .sort()
        .map(resourceName => {
            if (isEditing) {
                return (
                    <Field
                        key={`${name}.${resourceName}`}
                        name={`${name}.${resourceName}`}
                        resourceName={resourceName}
                        resourceToAccess={resourceToAccess}
                        isEditing={isEditing}
                        component={AccessField}
                    />
                );
            }
            return (
                <AccessField
                    key={`${name}.${resourceName}`}
                    resourceName={resourceName}
                    resourceToAccess={resourceToAccess}
                    isEditing={isEditing}
                />
            );
        });
};

TableRows.propTypes = {
    name: PropTypes.string.isRequired,
    resourceToAccess: PropTypes.shape({}).isRequired,
    isEditing: PropTypes.bool.isRequired
};

export default TableRows;
