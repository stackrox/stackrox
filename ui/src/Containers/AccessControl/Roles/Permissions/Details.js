import React from 'react';
import PropTypes from 'prop-types';

import PermissionsMatrix from 'Containers/AccessControl/Roles/Permissions/PermissionsMatrix/PermissionsMatrix';

const Details = props => {
    const { name, resourceToAccess } = props.role;
    return (
        <div className="w-full justify-between overflow-auto p-4">
            <div className="mb-4">
                <div className="py-2 text-base-600 font-700 text-lg">Role Name</div>
                <div>{name}</div>
            </div>
            <div>
                <PermissionsMatrix
                    name="resourceToAccess"
                    resourceToAccess={resourceToAccess}
                    isEditing={false}
                />
            </div>
        </div>
    );
};

Details.propTypes = {
    role: PropTypes.shape({
        name: PropTypes.string.isRequired,
        globalAccess: PropTypes.string.isRequired,
        resourceToAccess: PropTypes.shape({})
    }).isRequired
};

export default Details;
