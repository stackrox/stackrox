import React from 'react';
import PropTypes from 'prop-types';
import { gql } from '@apollo/client';
import queryService from 'utils/queryService';

import Query from 'Components/ThrowingQuery';
import Loader from 'Components/Loader';
import ViolationFindings from './ViolationFindings';

const QUERY = gql`
    query violationsInDeployment($query: String) {
        violations(query: $query) {
            id
            time
            policy {
                id
                enforcementActions
                categories
            }
            violations {
                message
            }
        }
    }
`;

const ViolationsAcrossThisDeployment = ({ deploymentID, policyID, message }) => {
    const variables = {
        query: queryService.objectToWhereClause({
            'Deployment ID': deploymentID,
            'Policy ID': policyID,
        }),
    };
    return (
        <Query query={QUERY} variables={variables}>
            {({ loading, data }) => {
                if (loading) {
                    return <Loader />;
                }
                if (!data) {
                    return null;
                }
                return <ViolationFindings data={data} message={message} />;
            }}
        </Query>
    );
};

ViolationsAcrossThisDeployment.propTypes = {
    deploymentID: PropTypes.string.isRequired,
    policyID: PropTypes.string.isRequired,
    message: PropTypes.string.isRequired,
};

export default ViolationsAcrossThisDeployment;
