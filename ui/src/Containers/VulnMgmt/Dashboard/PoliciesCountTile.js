import React, { useContext } from 'react';
import entityTypes from 'constants/entityTypes';
import { useQuery } from 'react-apollo';
import gql from 'graphql-tag';

import workflowStateContext from 'Containers/workflowStateContext';
import { generateURLTo } from 'modules/URLReadWrite';

import EntityTileLink from 'Components/EntityTileLink';

const POLICIES_COUNT_QUERY = gql`
    query policiesCount {
        policies {
            id
            fields {
                cve
                cvss {
                    scoreVersion: op
                    cvss: value
                }
            }
            alertCount
        }
    }
`;

const getURL = workflowState => {
    const url = generateURLTo(workflowState, entityTypes.POLICY);
    return url;
};

const PoliciesCountTile = () => {
    const { loading, data = {} } = useQuery(POLICIES_COUNT_QUERY);

    const { policies = [] } = data;

    const vulnPolicies = policies.filter(policy => {
        return policy.fields && (policy.fields.cve !== '' || policy.fields.cvss !== null);
    });

    const policyCount = vulnPolicies.length;
    const failingPoliciesCount = vulnPolicies.filter(policy => !!policy.alertCount).length;
    const failingPoliciesCountText = `(${failingPoliciesCount} failing)`;

    const workflowState = useContext(workflowStateContext);
    const url = getURL(workflowState);

    return (
        <EntityTileLink
            count={policyCount}
            entityType={entityTypes.POLICY}
            position="first"
            subText={failingPoliciesCountText}
            loading={loading}
            isError={!!failingPoliciesCount}
            url={url}
        />
    );
};

export default PoliciesCountTile;
