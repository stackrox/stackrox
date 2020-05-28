import React from 'react';
import PropTypes from 'prop-types';

import PermissionsMatrix from 'Containers/AccessControl/Roles/Permissions/PermissionsMatrix/PermissionsMatrix';

const Details = (props) => {
    const { name, resourceToAccess, authProviderName, username } = props.role;
    return (
        <div className="w-full justify-between overflow-auto p-4">
            <div className="mb-4 flex" data-testid="role-name-header">
                <div className="flex flex-col">
                    <div className="py-2 text-base-600 font-700 text-lg">
                        {username ? 'Username' : 'Role'}
                    </div>
                    <div>{username || name}</div>
                </div>
                {authProviderName && (
                    <div className="pl-4">
                        <div className="py-2 text-base-600 font-700 text-lg">Auth Provider</div>
                        <div>{authProviderName}</div>
                    </div>
                )}
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
        resourceToAccess: PropTypes.shape({}),
        authProviderName: PropTypes.string,
        username: PropTypes.string,
    }).isRequired,
};

export default Details;
