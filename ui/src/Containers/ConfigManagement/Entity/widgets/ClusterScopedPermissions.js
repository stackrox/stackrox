import React from 'react';
import Widget from 'Components/Widget';

import ScopedPermissions from './ScopedPermissions';

const ClusterScopedPermissionsWidget = ({ scopedPermissions, ...rest }) => {
    const clusterScopePermissions = scopedPermissions.filter(datum => datum.scope === 'Cluster');
    let content;
    if (clusterScopePermissions.length && clusterScopePermissions[0].permissions) {
        content = <ScopedPermissions permissions={clusterScopePermissions[0].permissions} />;
    }
    const header = `${
        clusterScopePermissions[0].permissions.length
    } Permissions across this cluster`;
    return (
        <Widget header={header} {...rest}>
            <div className="w-full">{content}</div>
        </Widget>
    );
};

export default ClusterScopedPermissionsWidget;
