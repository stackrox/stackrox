import React from 'react';
import entityTypes from 'constants/entityTypes';
import PropTypes from 'prop-types';
import { useTheme } from 'Containers/ThemeProvider';

import PageNotFound from 'Components/PageNotFound';
import ServiceAccount from './Entity/ServiceAccount';
import Secret from './Entity/Secret';
import Deployment from './Entity/Deployment/Deployment';
import Node from './Entity/Node';
import Cluster from './Entity/Cluster';
import Namespace from './Entity/Namespace';
import Role from './Entity/Role';
import Control from './Entity/Control';
import Image from './Entity/Image';
import Policy from './Entity/Policy';
import Subject from './Entity/Subject';

const entityComponentMap = {
    [entityTypes.SERVICE_ACCOUNT]: ServiceAccount,
    [entityTypes.ROLE]: Role,
    [entityTypes.SECRET]: Secret,
    [entityTypes.DEPLOYMENT]: Deployment,
    [entityTypes.CLUSTER]: Cluster,
    [entityTypes.NAMESPACE]: Namespace,
    [entityTypes.NODE]: Node,
    [entityTypes.CONTROL]: Control,
    [entityTypes.NODE]: Node,
    [entityTypes.IMAGE]: Image,
    [entityTypes.POLICY]: Policy,
    [entityTypes.SUBJECT]: Subject
};

const Entity = ({ entityType, entityId, entityListType, ...rest }) => {
    const { isDarkMode } = useTheme();
    const Component = entityComponentMap[entityType];
    if (!Component) return <PageNotFound resourceType={entityType} />;
    return (
        <div
            className={`w-full flex ${
                !isDarkMode && !entityListType ? 'bg-side-panel-wave min-h-full' : 'h-full'
            }`}
        >
            <Component id={entityId} entityListType={entityListType} {...rest} />
        </div>
    );
};

Entity.propTypes = {
    entityType: PropTypes.string.isRequired,
    entityListType: PropTypes.string,
    entityId: PropTypes.string.isRequired,
    query: PropTypes.shape({})
};
Entity.defaultProps = {
    query: null,
    entityListType: undefined
};

export default Entity;
