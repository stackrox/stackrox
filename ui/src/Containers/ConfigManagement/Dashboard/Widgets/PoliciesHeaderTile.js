import React from 'react';
import gql from 'graphql-tag';
import URLService from 'modules/URLService';
import { withRouter } from 'react-router-dom';
import ReactRouterPropTypes from 'react-router-prop-types';
import entityTypes from 'constants/entityTypes';
import Query from 'Components/ThrowingQuery';
import TileLink from 'Components/TileLink';
import queryService from 'modules/queryService';

const policiesQuery = gql`
    query policiesHeaderTile($query: String) {
        policies(query: $query) {
            id
            lifecycleStages
            alerts {
                state
                id
                violations {
                    message
                }
            }
        }
    }
`;

function processPoliciesData(data) {
    if (!data || !data.policies) return { totalPolicies: 0, hasViolations: false };

    const totalPolicies = data.policies.length;
    const hasViolations = !!data.policies.find(policy => {
        return policy.alerts.length > 0 && policy.alerts.find(alert => alert.state === 'ACTIVE');
    });
    return { totalPolicies, hasViolations };
}

const policiesHeaderTile = ({ match, location }) => {
    const policiesLink = URLService.getURL(match, location)
        .base(entityTypes.POLICY)
        .url();
    return (
        <Query
            query={policiesQuery}
            variables={{
                query: queryService.objectToWhereClause({ 'Lifecycle Stage': 'DEPLOY' })
            }}
        >
            {({ loading, data }) => {
                const { totalPolicies, hasViolations } = processPoliciesData(data);
                return (
                    <TileLink
                        value={totalPolicies}
                        isError={hasViolations}
                        caption="Policies"
                        to={policiesLink}
                        loading={loading}
                        className="rounded-l-sm border-r-0"
                    />
                );
            }}
        </Query>
    );
};

policiesHeaderTile.propTypes = {
    match: ReactRouterPropTypes.match.isRequired,
    location: ReactRouterPropTypes.location.isRequired
};

export default withRouter(policiesHeaderTile);
