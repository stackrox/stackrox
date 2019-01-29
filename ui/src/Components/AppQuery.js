import React from 'react';
import { Query } from 'react-apollo';
import Raven from 'raven-js';
import queryService from 'modules/queryService';
import PropTypes from 'prop-types';

const AppQuery = ({ children, params, componentType, ...rest }) => {
    const queryConfig = queryService.getQuery(params, componentType);
    if (!queryConfig) throw Error(`No query config found for ${componentType}`);

    return (
        <Query query={queryConfig.query} variables={queryConfig.variables} {...rest}>
            {queryResult => {
                if (queryResult.error) {
                    Raven.captureException(queryResult.error);
                }
                const results = {
                    ...queryResult
                };

                if (queryConfig.format && results.data) {
                    results.data = queryConfig.format(results.data);
                }

                return children(results);
            }}
        </Query>
    );
};

AppQuery.propTypes = {
    children: PropTypes.func.isRequired,
    params: PropTypes.shape({}).isRequired,
    componentType: PropTypes.string.isRequired
};

export default AppQuery;
