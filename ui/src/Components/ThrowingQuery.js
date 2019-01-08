import React from 'react';
import { Query } from 'react-apollo';

import Raven from 'raven-js';

const ThrowingQuery = ({ children, ...rest }) => (
    <Query {...rest}>
        {queryResult => {
            if (queryResult.error) {
                Raven.captureException(queryResult.error);
            }
            return children(queryResult);
        }}
    </Query>
);

export default ThrowingQuery;
