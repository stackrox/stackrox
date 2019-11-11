import React, { useContext } from 'react';
import PropTypes from 'prop-types';
import entityTypes from 'constants/entityTypes';
import { Link } from 'react-router-dom';
import { useQuery } from 'react-apollo';
import gql from 'graphql-tag';
import queryService from 'modules/queryService';
import { severityLabels } from 'messages/common';
import sortBy from 'lodash/sortBy';

import workflowStateContext from 'Containers/workflowStateContext';

import Button from 'Components/Button';
import Loader from 'Components/Loader';
import Widget from 'Components/Widget';
import LabeledBarGraph from 'Components/visuals/LabeledBarGraph';
import NoResultsMessage from 'Components/NoResultsMessage';

const FREQUENTLY_VIOLATED_POLICIES = gql`
    query frequentlyViolatedPolicies($query: String) {
        results: policies(query: $query) {
            id
            name
            enforcementActions
            severity
            alertCount
        }
    }
`;

const ViewAllButton = ({ url }) => {
    return (
        <Link to={url} className="no-underline">
            <Button className="btn-sm btn-base" type="button" text="View All" />
        </Link>
    );
};

const processData = (data, workflowState, limit) => {
    const results = sortBy(data.results, ['alertCount']).slice(-limit); // @TODO: Remove when we have pagination on Policies
    return results
        .filter(datum => datum.alertCount)
        .map(({ id, name, enforcementActions, severity, alertCount }) => {
            const url = workflowState.pushRelatedEntity(entityTypes.POLICY, id).toUrl();
            const isEnforced = enforcementActions.length ? 'Yes' : 'No';
            return {
                x: alertCount,
                y: `${name} / Enforced: ${isEnforced} / Severity: ${severityLabels[severity]}`,
                url
            };
        });
};

const FrequentlyViolatedPolicies = ({ entityContext, limit }) => {
    const { loading, data = {} } = useQuery(FREQUENTLY_VIOLATED_POLICIES, {
        variables: {
            query: `${queryService.entityContextToQueryString(entityContext)}+
            ${queryService.objectToWhereClause({ Category: 'Vulnerability Management' })}`
        }
    });

    let content = <Loader />;

    const workflowState = useContext(workflowStateContext);
    if (!loading) {
        const processedData = processData(data, workflowState, limit);

        if (!processedData || processedData.length === 0) {
            content = (
                <NoResultsMessage
                    message="No deployments with policy violations found"
                    className="p-6"
                    icon="info"
                />
            );
        } else {
            content = <LabeledBarGraph data={processedData} title="Failing Deployments" />;
        }
    }

    const viewAllURL = workflowState
        .pushList(entityTypes.POLICY)
        .setSort([{ id: 'policyStatus', desc: false }, { id: 'severity', desc: false }])
        .toUrl();

    return (
        <Widget
            className="h-full pdf-page"
            bodyClassName="px-2"
            header="Frequently Violated Policies"
            headerComponents={<ViewAllButton url={viewAllURL} />}
        >
            {content}
        </Widget>
    );
};

FrequentlyViolatedPolicies.propTypes = {
    entityContext: PropTypes.shape({}),
    limit: PropTypes.number
};

FrequentlyViolatedPolicies.defaultProps = {
    entityContext: {},
    limit: 9
};

export default FrequentlyViolatedPolicies;
