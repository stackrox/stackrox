import React from 'react';
import PropTypes from 'prop-types';

const flattenPermissions = (scopedPermissions) => {
    let permissions = [];
    scopedPermissions.forEach((datum) => {
        permissions = [...permissions, ...datum.permissions];
    });
    return permissions;
};

const createPermissionCountsMapping = (permissions) => {
    const permissionsMap = permissions.reduce((acc, curr) => {
        acc[curr.key] = [...(acc[curr.key] || []), ...curr.values];
        return acc;
    }, {});
    return permissionsMap;
};

const getPermissionKey = (key) => {
    if (key === '*') {
        return 'all';
    }
    return key;
};

const getPermissionValues = (values) => {
    if (!values || !values.length) {
        return '';
    }
    if (values.length > 1) {
        return `(${values.length})`;
    }
    return values[0];
};

const getPermissionCounts = (scopedPermissions) => {
    if (!scopedPermissions.length) {
        return 'No Permissions';
    }
    const permissions = flattenPermissions(scopedPermissions);
    const permissionsMap = createPermissionCountsMapping(permissions);
    const result = Object.keys(permissionsMap)
        .sort()
        .reduce((acc, key) => {
            const values = permissionsMap[key];
            const permissionKey = getPermissionKey(key);
            const permissionValues = getPermissionValues(values);
            return `${acc}${acc !== '' ? ',' : ''} ${permissionKey} ${permissionValues}`;
        }, '');
    return result;
};

const PermissionCounts = ({ scopedPermissions }) => {
    const permissionCounts = getPermissionCounts(scopedPermissions);
    return <span>{permissionCounts}</span>;
};

PermissionCounts.propTypes = {
    scopedPermissions: PropTypes.arrayOf(
        PropTypes.shape({
            scope: PropTypes.string,
            permissions: PropTypes.arrayOf(
                PropTypes.shape({
                    key: PropTypes.string.isRequired,
                    values: PropTypes.arrayOf(PropTypes.string),
                })
            ),
        })
    ).isRequired,
};

export default PermissionCounts;
