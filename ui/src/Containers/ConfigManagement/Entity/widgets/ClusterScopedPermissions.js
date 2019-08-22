import React from 'react';
import Widget from 'Components/Widget';
import NoResultsMessage from 'Components/NoResultsMessage';
import ScopedPermissions from './ScopedPermissions';

const ClusterScopedPermissionsWidget = ({ scopedPermissions, ...rest }) => {
    const clusterScopePermissions = scopedPermissions.filter(datum => datum.scope === 'Cluster');
    let content = null;
    const permissions = clusterScopePermissions.reduce((acc, curr) => {
        return [...acc, ...curr.permissions];
    }, []);
    content = <ScopedPermissions permissions={permissions} />;
    if (permissions.length === 0)
        content = <NoResultsMessage message="No permissions available" className="p-6 shadow" />;
    const header = `${permissions.length > 0 ? permissions.length : ''} Permissions across ${
        clusterScopePermissions.length
    } cluster`;
    return (
        <Widget header={header} {...rest}>
            <div className="w-full">{content}</div>
        </Widget>
    );
};

export default ClusterScopedPermissionsWidget;
