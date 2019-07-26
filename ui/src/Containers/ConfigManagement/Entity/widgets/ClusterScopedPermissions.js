import React from 'react';
import Widget from 'Components/Widget';

import ScopedPermissions from './ScopedPermissions';

const ClusterScopedPermissionsWidget = ({ scopedPermissions, ...rest }) => {
    const clusterScopePermissions = scopedPermissions.filter(datum => datum.scope === 'Cluster');
    let content = null;
    const permissions = clusterScopePermissions.reduce((acc, curr) => {
        return [...acc, ...curr.permissions];
    }, []);
    content = <ScopedPermissions permissions={permissions} />;
    const header = `${permissions.length} Permissions across ${
        clusterScopePermissions.length
    } cluster`;
    return (
        <Widget header={header} {...rest}>
            <div className="w-full">{content}</div>
        </Widget>
    );
};

export default ClusterScopedPermissionsWidget;
