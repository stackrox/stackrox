import React, { useContext } from 'react';
import PropTypes from 'prop-types';
import { gql, useQuery } from '@apollo/client';
import sortBy from 'lodash/sortBy';
import { format } from 'date-fns';

import workflowStateContext from 'Containers/workflowStateContext';
import ViewAllButton from 'Components/ViewAllButton';
import Loader from 'Components/Loader';
import Widget from 'Components/Widget';
import LabeledBarGraph from 'Components/visuals/LabeledBarGraph';
import HoverHintListItem from 'Components/visuals/HoverHintListItem';
import NoResultsMessage from 'Components/NoResultsMessage';
import dateTimeFormat from 'constants/dateTimeFormat';
import entityTypes from 'constants/entityTypes';
import { resourceLabels, severityLabels } from 'messages/common';
import { checkForPermissionErrorMessage } from 'utils/permissionUtils';
import queryService from 'utils/queryService';
import { policySortFields } from 'constants/sortFields';

const FREQUENTLY_VIOLATED_POLICIES = gql`
    query frequentlyViolatedPolicies($query: String) {
        results: policies(query: $query) {
            id
            name
            enforcementActions
            severity
            alertCount
            categories
            description
            latestViolation
        }
    }
`;

const processData = (data, workflowState, limit) => {
    const results = sortBy(data.results, ['alertCount']).slice(-limit); // @TODO: Remove when we have pagination on Policies
    return results
        .filter((datum) => datum.alertCount)
        .map((datum) => {
            const {
                id,
                name,
                description,
                enforcementActions,
                severity,
                alertCount,
                latestViolation,
                categories,
            } = datum;
            const url = workflowState.pushRelatedEntity(entityTypes.POLICY, id).toUrl();
            const isEnforced = enforcementActions.length ? 'Yes' : 'No';
            const categoriesStr = categories.join(', ');

            const tooltipBody = (
                <ul className="flex-1 border-base-300 overflow-hidden">
                    <HoverHintListItem key="categories" label="Category" value={categoriesStr} />
                    <HoverHintListItem key="description" label="Description" value={description} />
                    <HoverHintListItem
                        key="latestViolation"
                        label="Last violated"
                        value={format(latestViolation, dateTimeFormat)}
                    />
                </ul>
            );

            return {
                // Alert count cann be treated as deployment count only under the assumption that
                // Vulnerability Management policies do not have runtime violations.
                x: alertCount,
                y: `${name} / Enforced: ${isEnforced} / Severity: ${severityLabels[severity]}`,
                url,
                hint: { title: name, body: tooltipBody },
            };
        });
};

const FrequentlyViolatedPolicies = ({ entityContext, limit }) => {
    // combine any given scope (empty on dashboards) with a Policy Category filter of "Vulnerability Management"
    const entityContextObject = queryService.entityContextToQueryObject(entityContext);
    const queryObject = { ...entityContextObject, Category: 'Vulnerability Management' };
    const query = queryService.objectToWhereClause(queryObject); // get final gql query string

    const {
        loading,
        data = {},
        error,
    } = useQuery(FREQUENTLY_VIOLATED_POLICIES, {
        variables: {
            query,
        },
    });

    let content = <Loader />;
    const workflowState = useContext(workflowStateContext);
    if (!loading) {
        if (error) {
            const defaultMessage = `An error occurred in retrieving ${resourceLabels[entityContext]}s. Please refresh the page. If this problem continues, please contact support.`;

            const parsedMessage = checkForPermissionErrorMessage(error, defaultMessage);

            content = <NoResultsMessage message={parsedMessage} className="p-3" icon="warn" />;
        } else if (data) {
            const processedData = processData(data, workflowState, limit);

            if (!processedData || processedData.length === 0) {
                content = (
                    <NoResultsMessage
                        message="No deployments with policy violations found"
                        className="p-3"
                        icon="info"
                    />
                );
            } else {
                content = <LabeledBarGraph data={processedData} title="Failing Deployments" />;
            }
        }
    }

    const viewAllURL = workflowState
        .pushList(entityTypes.POLICY)
        .setSort([
            // @TODO to uncomment once Policy Status field is sortable on backend
            // { id: policySortFields.POLICY_STATUS, desc: false },
            { id: policySortFields.SEVERITY, desc: true },
        ])
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
    limit: PropTypes.number,
};

FrequentlyViolatedPolicies.defaultProps = {
    entityContext: {},
    limit: 7,
};

export default FrequentlyViolatedPolicies;
