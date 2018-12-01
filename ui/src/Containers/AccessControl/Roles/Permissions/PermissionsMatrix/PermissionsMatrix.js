import React from 'react';
import PropTypes from 'prop-types';
import TableRows from 'Containers/AccessControl/Roles/Permissions/PermissionsMatrix/TableRows';

const PermissionsMatrix = props => (
    <table className="w-full overflow-auto">
        <thead>
            <tr className="border-b border-base-300">
                <th className="p-4" />
                <th className="p-4">Read</th>
                <th className="p-4">Write</th>
                {props.isEditing && <th className="p-4">Edit</th>}
            </tr>
        </thead>
        <tbody>
            <TableRows
                name={props.name}
                resourceToAccess={props.resourceToAccess}
                isEditing={props.isEditing}
            />
        </tbody>
    </table>
);

PermissionsMatrix.propTypes = {
    name: PropTypes.string.isRequired,
    resourceToAccess: PropTypes.shape({}),
    isEditing: PropTypes.bool.isRequired
};

PermissionsMatrix.defaultProps = {
    resourceToAccess: {}
};

export default PermissionsMatrix;
