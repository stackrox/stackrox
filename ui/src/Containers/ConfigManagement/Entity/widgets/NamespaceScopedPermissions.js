import React from 'react';
import PropTypes from 'prop-types';
import Widget from 'Components/Widget';
import CollapsibleRow from 'Components/CollapsibleRow';
import ScopedPermissions from './ScopedPermissions';

const PermissionsCounts = ({ permissions }) => {
    const permissionsCounts = permissions.reduce((acc, curr) => {
        acc[curr.key] = (acc[curr.key] || 0) + curr.values.length;
        return acc;
    }, {});
    const result = Object.keys(permissionsCounts).map(key => {
        const value = permissionsCounts[key];
        return (
            <li className="flex mr-2" key={key}>
                {key} ({value})
            </li>
        );
    });
    return <ul className="flex text-sm list-reset capitalize">{result}</ul>;
};

PermissionsCounts.propTypes = {
    permissions: PropTypes.arrayOf(
        PropTypes.shape({
            key: PropTypes.string,
            values: PropTypes.arrayOf(PropTypes.string)
        })
    ).isRequired
};

const NamespaceScopedPermissions = ({ scopedPermissions, ...rest }) => {
    const namespaceScopePermissions = scopedPermissions.filter(datum => datum.scope !== 'Cluster');
    const namespaceGroups = namespaceScopePermissions.map(({ scope, permissions }) => {
        const groupHeader = (
            <div className="flex flex-1">
                <div className="flex flex-1">{scope}</div>
                <div>
                    <PermissionsCounts permissions={permissions} />
                </div>
            </div>
        );
        const group = (
            <CollapsibleRow key={scope} header={groupHeader}>
                <ScopedPermissions permissions={permissions} />
            </CollapsibleRow>
        );
        return group;
    });
    const header = `Permissions across ${namespaceGroups.length} namespaces`;
    return (
        <Widget header={header} {...rest}>
            <div className="w-full mx-4">{namespaceGroups}</div>
        </Widget>
    );
};

export default NamespaceScopedPermissions;
