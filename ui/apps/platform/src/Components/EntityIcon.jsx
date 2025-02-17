import React from 'react';
import PropTypes from 'prop-types';
import entityTypes from 'constants/entityTypes';
import cluster from 'images/side-panel-icons/cluster.svg';
import deployment from 'images/side-panel-icons/deployment.svg';
import group from 'images/side-panel-icons/group.svg';
import image from 'images/side-panel-icons/image.svg';
import cve from 'images/side-panel-icons/cve.svg';
import component from 'images/side-panel-icons/image-layer.svg';
import namespace from 'images/side-panel-icons/namespace.svg';
import node from 'images/side-panel-icons/node.svg';
import policy from 'images/side-panel-icons/policy.svg';
import role from 'images/side-panel-icons/role.svg';
import secrets from 'images/side-panel-icons/secrets.svg';
import serviceAccount from 'images/side-panel-icons/service-account.svg';
import control from 'images/side-panel-icons/control.svg';

const imageMap = {
    [entityTypes.CLUSTER]: cluster,
    [entityTypes.DEPLOYMENT]: deployment,
    [entityTypes.SUBJECT]: group,
    [entityTypes.IMAGE]: image,
    [entityTypes.COMPONENT]: component,
    [entityTypes.NODE_COMPONENT]: component,
    [entityTypes.IMAGE_COMPONENT]: component,
    [entityTypes.CVE]: cve,
    [entityTypes.IMAGE_CVE]: cve,
    [entityTypes.NODE_CVE]: cve,
    [entityTypes.CLUSTER_CVE]: cve,
    [entityTypes.NAMESPACE]: namespace,
    [entityTypes.NODE]: node,
    [entityTypes.POLICY]: policy,
    [entityTypes.ROLE]: role,
    [entityTypes.SECRET]: secrets,
    [entityTypes.SERVICE_ACCOUNT]: serviceAccount,
    [entityTypes.CONTROL]: control,
};

const EntityIcon = ({ className, entityType }) => (
    <img
        className={className}
        src={imageMap[entityType]}
        alt={`${entityType} entity`}
        data-testid="entity-icon"
    />
);

EntityIcon.propTypes = {
    className: PropTypes.string,
    entityType: PropTypes.string.isRequired,
};

EntityIcon.defaultProps = {
    className: '',
};

export default EntityIcon;
