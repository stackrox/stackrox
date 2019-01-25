import React from 'react';
import { Query } from 'react-apollo';
import Raven from 'raven-js';
import queryService from 'modules/queryService';
import PropTypes from 'prop-types';

const AppQuery = ({ children, pageId, params, componentType, ...rest }) => {
    const queryConfig = queryService.getQuery(pageId, params, componentType);
    if (!queryConfig)
        throw Error(`No query config found for ${componentType}, ${JSON.stringify(pageId)}`);

    return (
        <Query query={queryConfig.query} variables={queryConfig.variables} {...rest}>
            {queryResult => {
                if (queryResult.error) {
                    Raven.captureException(queryResult.error);
                }
                return children(queryResult);
            }}
        </Query>
    );
};

AppQuery.propTypes = {
    children: PropTypes.func.isRequired,
    pageId: PropTypes.shape({}).isRequired,
    params: PropTypes.shape({}).isRequired,
    componentType: PropTypes.string.isRequired
};

export default AppQuery;
