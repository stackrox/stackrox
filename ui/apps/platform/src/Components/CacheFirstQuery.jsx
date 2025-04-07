import React from 'react';
import Query from './ThrowingQuery';

const CacheFirstQuery = ({ children, ...rest }) => (
    <Query fetchPolicy="cache-first" {...rest}>
        {(queryResult) => {
            return children(queryResult);
        }}
    </Query>
);

export default CacheFirstQuery;
