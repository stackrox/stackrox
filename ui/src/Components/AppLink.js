import React from 'react';
import PropTypes from 'prop-types';
import { Link, withRouter, generatePath } from 'react-router-dom';
import qs from 'qs';
import { resourceTypes } from 'constants/entityTypes';
import pageTypes from 'constants/pageTypes';
import contextTypes from 'constants/contextTypes';
import { nestedCompliancePaths } from '../routePaths';

function getPath(context, pageType, urlParams) {
    const isResource = Object.values(resourceTypes).includes(urlParams.entityType);
    const pathMap = {
        [contextTypes.COMPLIANCE]: {
            [pageTypes.DASHBOARD]: nestedCompliancePaths.DASHBOARD,
            [pageTypes.ENTITY]: isResource
                ? nestedCompliancePaths.RESOURCE
                : nestedCompliancePaths.CONTROL,
            [pageTypes.LIST]: nestedCompliancePaths.LIST
        }
    };

    const contextData = pathMap[context];
    if (!contextData) return null;

    const path = contextData[pageType];
    if (!path) return null;

    return generatePath(path, urlParams);
}

const AppLink = ({ context, pageType, entityType, params, children, staticContext, ...rest }) => {
    const { query, ...urlParams } = params;
    const to = {
        pathname: getPath(context, pageType, urlParams),
        search: query ? qs.stringify(query, { addQueryPrefix: true }) : null
    };

    return (
        <Link to={to} {...rest}>
            {children}
        </Link>
    );
};

AppLink.propTypes = {
    context: PropTypes.string.isRequired,
    pageType: PropTypes.string.isRequired,
    entityType: PropTypes.string.isRequired,
    params: PropTypes.shape({}).isRequired
};

export default withRouter(AppLink);
