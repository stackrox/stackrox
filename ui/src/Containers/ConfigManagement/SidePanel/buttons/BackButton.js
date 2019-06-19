import React from 'react';
import ReactRouterPropTypes from 'react-router-prop-types';
import PropTypes from 'prop-types';
import entityTypes from 'constants/entityTypes';
import { withRouter, Link } from 'react-router-dom';
import URLService from 'modules/URLService';

import { ArrowLeft } from 'react-feather';
import cluster from 'images/side-panel-icons/cluster.svg';
import deployment from 'images/side-panel-icons/deployment.svg';
import group from 'images/side-panel-icons/group.svg';
import image from 'images/side-panel-icons/image.svg';
import namespace from 'images/side-panel-icons/namespace.svg';
import node from 'images/side-panel-icons/node.svg';
import policy from 'images/side-panel-icons/policy.svg';
import role from 'images/side-panel-icons/role.svg';
import secrets from 'images/side-panel-icons/secrets.svg';
import serviceAccount from 'images/side-panel-icons/service-account.svg';

const imageMap = {
    [entityTypes.CLUSTER]: cluster,
    [entityTypes.DEPLOYMENT]: deployment,
    [entityTypes.SUBJECT]: group,
    [entityTypes.IMAGE]: image,
    [entityTypes.NAMESPACE]: namespace,
    [entityTypes.NODE]: node,
    [entityTypes.POLICY]: policy,
    [entityTypes.ROLE]: role,
    [entityTypes.SECRET]: secrets,
    [entityTypes.SERVICE_ACCOUNT]: serviceAccount
};

const BackButton = ({ match, location, entityType1, entityListType2, entityId2 }) => {
    if (entityListType2 || entityId2) {
        const link = URLService.getURL(match, location)
            .pop()
            .url();
        return (
            <Link
                className="flex items-center justify-center text-base-600 border-r border-base-300 px-4 mr-4 h-full hover:bg-primary-200 w-16"
                to={link}
            >
                <ArrowLeft className="h-6 w-6 text-600" />
            </Link>
        );
    }
    return (
        <img
            className="flex items-center justify-center border-r border-base-300 px-4 mr-4 h-full w-16"
            src={imageMap[entityType1]}
            alt={`${entityType1} entity`}
        />
    );
};

BackButton.propTypes = {
    match: ReactRouterPropTypes.match.isRequired,
    location: ReactRouterPropTypes.location.isRequired,
    entityType1: PropTypes.string,
    entityListType2: PropTypes.string,
    entityId2: PropTypes.string
};

BackButton.defaultProps = {
    entityType1: null,
    entityListType2: null,
    entityId2: null
};

export default withRouter(BackButton);
