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

const processData = (data, workflowState) => {
    const results = sortBy(data.results, [datum => datum.alertCount]).splice(-9); // @TODO: Remove when we have pagination on Policies
    return results.map(({ id, name, enforcementActions, severity, alertCount }) => {
        const url = workflowState.pushRelatedEntity(entityTypes.POLICY, id).toUrl();
        const isEnforced = enforcementActions.length ? 'Yes' : 'No';
        return {
            x: alertCount,
            y: `${name} / Enforced: ${isEnforced} / Severity: ${severityLabels[severity]}`,
            url
        };
    });
};

const FrequentlyViolatedPolicies = ({ entityContext }) => {
    const { loading, data = {} } = useQuery(FREQUENTLY_VIOLATED_POLICIES, {
        variables: {
            query: queryService.entityContextToQueryString(entityContext)
        }
    });

    let content = <Loader />;

    const workflowState = useContext(workflowStateContext);
    if (!loading) {
        const processedData = processData(data, workflowState);

        content = <LabeledBarGraph data={processedData} title="Failing Deployments" />;
    }

    return (
        <Widget
            className="h-full pdf-page"
            bodyClassName="px-2"
            header="Frequently Violated Policies"
            headerComponents={
                <ViewAllButton url={workflowState.pushList(entityTypes.POLICY).toUrl()} />
            }
        >
            {content}
        </Widget>
    );
};

FrequentlyViolatedPolicies.propTypes = {
    entityContext: PropTypes.shape({})
};

FrequentlyViolatedPolicies.defaultProps = {
    entityContext: {}
};

export default FrequentlyViolatedPolicies;
