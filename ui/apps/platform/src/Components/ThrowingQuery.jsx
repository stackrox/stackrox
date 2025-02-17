import React from 'react';
import { Query } from '@apollo/client/react/components';
import PropTypes from 'prop-types';
import Raven from 'raven-js';
import * as Icons from 'react-feather';

const GraphQLError = ({ error }) => (
    <div
        className="flex h-full w-1/2 m-auto items-center justify-center bg-base-100 text-base-600"
        data-testid="graphql-error"
    >
        <div className="flex items-center justify-center">
            <Icons.XSquare size="48" />
        </div>
        <div className="pl-2">
            <div className="text-2xl">An Error has occurred</div>
            <div className="pt-2">{error.message}</div>
        </div>
    </div>
);

GraphQLError.propTypes = {
    error: PropTypes.shape({ message: PropTypes.string.isRequired }).isRequired,
};

const ThrowingQuery = ({ children, ...rest }) => (
    <Query {...rest}>
        {({ error, ...queryResult }) => {
            if (error) {
                Raven.captureException(error);
                if (process.env.NODE_ENV === 'development') {
                    return <GraphQLError error={error} />;
                }
            }
            return children({ error, ...queryResult });
        }}
    </Query>
);

export default ThrowingQuery;
