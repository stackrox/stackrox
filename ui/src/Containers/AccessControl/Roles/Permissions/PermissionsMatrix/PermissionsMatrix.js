import React from 'react';
import PropTypes from 'prop-types';
import TableRows from 'Containers/AccessControl/Roles/Permissions/PermissionsMatrix/TableRows';

const PermissionsMatrix = props => (
    <table className="w-full overflow-auto mt-6">
        <thead>
            <tr className="border-b border-base-300">
                <th className="text-lg text-left p-2 pl-0">Permissions</th>
                <th className="text-lg p-2">Read</th>
                <th className="text-lg p-2">Write</th>
                {props.isEditing && <th className="text-lg p-2">Edit role</th>}
            </tr>
        </thead>
        <tbody className="p-3">
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
    resourceToAccess: PropTypes.shape({}).isRequired,
    isEditing: PropTypes.bool.isRequired
};

export default PermissionsMatrix;
