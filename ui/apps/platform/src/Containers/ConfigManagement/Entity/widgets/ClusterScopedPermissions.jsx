import React from 'react';
import PropTypes from 'prop-types';
import Widget from 'Components/Widget';
import NoResultsMessage from 'Components/NoResultsMessage';
import CollapsibleRow from 'Components/CollapsibleRow';
import ScopedPermissions from './ScopedPermissions';

const getContent = (scopedPermissions) => {
    const clusterScopePermissions = scopedPermissions.filter((datum) => datum.scope === 'Cluster');
    let content = null;
    const permissions = clusterScopePermissions.reduce((acc, curr) => {
        return [...acc, ...curr.permissions];
    }, []);
    content = <ScopedPermissions permissions={permissions} />;
    if (permissions.length === 0) {
        content = <NoResultsMessage message="No permissions available" className="p-3 shadow" />;
    }
    return content;
};

const getGroupedContent = (scopedPermissionsByCluster) => {
    return scopedPermissionsByCluster.map(({ clusterId, clusterName, scopedPermissions }) => {
        const groupHeader = clusterName;
        const scopedPermissionsContent = getContent(scopedPermissions);
        const group = (
            <CollapsibleRow key={clusterId} header={groupHeader}>
                {scopedPermissionsContent}
            </CollapsibleRow>
        );
        return group;
    });
};

const ClusterScopedPermissions = ({ scopedPermissionsByCluster, ...rest }) => {
    let content = null;

    if (!scopedPermissionsByCluster || !scopedPermissionsByCluster.length) {
        content = <NoResultsMessage message="No permissions available" className="p-3 shadow" />;
    } else if (scopedPermissionsByCluster.length > 1) {
        content = getGroupedContent(scopedPermissionsByCluster);
    } else {
        const { scopedPermissions } = scopedPermissionsByCluster[0];
        content = getContent(scopedPermissions);
    }

    const header =
        scopedPermissionsByCluster.length > 1
            ? 'Cluster Permissions across all clusters'
            : `Cluster Permissions in "${
                  scopedPermissionsByCluster[0] && scopedPermissionsByCluster[0].clusterName
              }" cluster`;

    return (
        <Widget header={header} {...rest}>
            <div className="w-full">{content}</div>
        </Widget>
    );
};

ClusterScopedPermissions.propTypes = {
    scopedPermissionsByCluster: PropTypes.arrayOf(
        PropTypes.shape({
            clusterId: PropTypes.string.isRequired,
            clusterName: PropTypes.string.isRequired,
            scopedPermissions: PropTypes.arrayOf(PropTypes.shape({})),
        })
    ),
};

ClusterScopedPermissions.defaultProps = {
    scopedPermissionsByCluster: [],
};

export default ClusterScopedPermissions;
