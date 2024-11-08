import React from 'react';
import PropTypes from 'prop-types';
import Widget from 'Components/Widget';
import NoResultsMessage from 'Components/NoResultsMessage';
import CollapsibleRow from 'Components/CollapsibleRow';
import ScopedPermissions from './ScopedPermissions';

const PermissionsCounts = ({ permissions }) => {
    const permissionsCounts = permissions.reduce((acc, curr) => {
        acc[curr.key] = (acc[curr.key] || 0) + curr.values.length;
        return acc;
    }, {});
    const result = Object.keys(permissionsCounts).map((key) => {
        const value = permissionsCounts[key];
        return (
            <li className="flex mr-2" key={key}>
                {key} ({value})
            </li>
        );
    });
    return <ul className="flex text-sm capitalize">{result}</ul>;
};

PermissionsCounts.propTypes = {
    permissions: PropTypes.arrayOf(
        PropTypes.shape({
            key: PropTypes.string,
            values: PropTypes.arrayOf(PropTypes.string),
        })
    ).isRequired,
};

const filterNamespaceScopePermissions = (datum) => datum.scope !== 'Cluster';

const getContent = (scopedPermissions) => {
    const namespaceScopePermissions = scopedPermissions.filter(filterNamespaceScopePermissions);
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
    if (!namespaceGroups.length) {
        return null;
    }
    const content = namespaceGroups;
    return content;
};

const getGroupedContent = (scopedPermissionsByCluster) => {
    return scopedPermissionsByCluster
        .filter(
            ({ scopedPermissions }) =>
                scopedPermissions.filter(filterNamespaceScopePermissions).length
        )
        .map(({ clusterId, clusterName, scopedPermissions }) => {
            const groupHeader = clusterName;
            const scopedPermissionsContent = getContent(scopedPermissions);
            if (!scopedPermissionsContent) {
                return null;
            }
            const group = (
                <CollapsibleRow key={clusterId} header={groupHeader}>
                    <div className="pl-4">{scopedPermissionsContent}</div>
                </CollapsibleRow>
            );
            return group;
        });
};

const NamespaceScopedPermissions = ({ scopedPermissionsByCluster, ...rest }) => {
    let content = null;

    if (!scopedPermissionsByCluster || !scopedPermissionsByCluster.length) {
        content = <NoResultsMessage message="No permissions available" className="p-3 shadow" />;
    } else if (scopedPermissionsByCluster.length > 1) {
        content = getGroupedContent(scopedPermissionsByCluster);
    } else {
        const { scopedPermissions } = scopedPermissionsByCluster[0];
        content = getContent(scopedPermissions);
    }

    if (!content || !content.length) {
        content = <NoResultsMessage message="No permissions available" className="p-3 shadow" />;
    }

    const header =
        scopedPermissionsByCluster.length > 1
            ? 'Namespace Permissions across all clusters'
            : `Namespace Permissions in "${
                  scopedPermissionsByCluster[0] && scopedPermissionsByCluster[0].clusterName
              }" cluster`;

    return (
        <Widget header={header} {...rest}>
            <div className="w-full">{content}</div>
        </Widget>
    );
};

NamespaceScopedPermissions.propTypes = {
    scopedPermissionsByCluster: PropTypes.arrayOf(
        PropTypes.shape({
            clusterId: PropTypes.string.isRequired,
            clusterName: PropTypes.string.isRequired,
            scopedPermissions: PropTypes.arrayOf(PropTypes.shape({})),
        })
    ),
};

NamespaceScopedPermissions.defaultProps = {
    scopedPermissionsByCluster: [],
};

export default NamespaceScopedPermissions;
