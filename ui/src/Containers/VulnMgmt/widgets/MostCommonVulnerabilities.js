import React, { useContext } from 'react';
import PropTypes from 'prop-types';
import { useQuery } from 'react-apollo';
import gql from 'graphql-tag';
import sortBy from 'lodash/sortBy';
import { format } from 'date-fns';

import workflowStateContext from 'Containers/workflowStateContext';
import ViewAllButton from 'Components/ViewAllButton';
import Loader from 'Components/Loader';
import Widget from 'Components/Widget';
import LabeledBarGraph from 'Components/visuals/LabeledBarGraph';
import NoResultsMessage from 'Components/NoResultsMessage';
import queryService from 'modules/queryService';
import dateTimeFormat from 'constants/dateTimeFormat';
import entityTypes from 'constants/entityTypes';
import { cveSortFields } from 'constants/sortFields';
import { WIDGET_PAGINATION_START_OFFSET } from 'constants/workflowPages.constants';

const MOST_COMMON_VULNERABILITIES = gql`
    query mostCommonVulnerabilities($query: String, $vulnPagination: Pagination) {
        results: vulnerabilities(query: $query, pagination: $vulnPagination) {
            id
            cve
            cvss
            scoreVersion
            isFixable
            deploymentCount
            summary
            imageCount
            lastScanned
        }
    }
`;

const processData = (data, workflowState, limit) => {
    const results = sortBy(data.results, ['deploymentCount', 'cvss']).slice(-limit); // @TODO: Remove when we have pagination on Vulnerabilities
    return results.map(
        ({
            id,
            cve,
            cvss,
            summary,
            scoreVersion,
            isFixable,
            deploymentCount,
            imageCount,
            lastScanned
        }) => {
            const url = workflowState.pushRelatedEntity(entityTypes.CVE, id).toUrl();
            const tooltipTitle = lastScanned ? format(lastScanned, dateTimeFormat) : 'N/A';
            const tooltipBody = (
                <ul className="flex-1 border-base-300 overflow-hidden">
                    <li className="py-1 flex flex-col" key="description">
                        <span className="text-base-600 font-700 mr-2">Description:</span>
                        <span className="font-600">{summary}</span>
                    </li>
                    <li className="py-1 flex flex-col" key="latestViolation">
                        <span className="text-base-600 font-700 mr-2">Impact:</span>
                        <span className="font-600">{`${deploymentCount} deployments, ${imageCount} images`}</span>
                    </li>
                </ul>
            );
            const tooltipFooter = `Scored using CVSS ${scoreVersion}`;

            return {
                x: deploymentCount,
                y: `${cve} / CVSS: ${cvss.toFixed(1)} (${scoreVersion}) ${
                    isFixable ? ' / Fixable' : ''
                }`,
                url,
                hint: {
                    title: tooltipTitle,
                    body: tooltipBody,
                    footer: tooltipFooter
                }
            };
        }
    );
};

const MostCommonVulnerabilities = ({ entityContext, search, limit }) => {
    const entityContextObject = queryService.entityContextToQueryObject(entityContext); // deals with BE inconsistency

    const queryObject = { ...entityContextObject, ...search }; // Combine entity context and search
    const query = queryService.objectToWhereClause(queryObject); // get final gql query string

    const { loading, data = {} } = useQuery(MOST_COMMON_VULNERABILITIES, {
        variables: {
            query,
            vulnPagination: queryService.getPagination(
                {
                    id: cveSortFields.DEPLOYMENT_COUNT,
                    desc: true
                },
                WIDGET_PAGINATION_START_OFFSET,
                limit
            )
        }
    });

    let content = <Loader />;

    const workflowState = useContext(workflowStateContext);
    if (!loading) {
        const processedData = processData(data, workflowState, limit);
        if (!processedData || processedData.length === 0) {
            content = (
                <NoResultsMessage message="No vulnerabilities found" className="p-3" icon="info" />
            );
        } else {
            content = <LabeledBarGraph data={processedData} title="Deployments" />;
        }
    }

    const viewAllURL = workflowState
        .pushList(entityTypes.CVE)
        .setSort([
            { id: cveSortFields.DEPLOYMENT_COUNT, desc: true },
            { id: cveSortFields.CVSS_SCORE, desc: true }
        ])
        .toUrl();

    return (
        <Widget
            className="h-full pdf-page"
            header="Most Common Vulnerabilities"
            headerComponents={<ViewAllButton url={viewAllURL} />}
        >
            {content}
        </Widget>
    );
};

MostCommonVulnerabilities.propTypes = {
    entityContext: PropTypes.shape({}),
    search: PropTypes.shape({}),
    limit: PropTypes.number
};

MostCommonVulnerabilities.defaultProps = {
    entityContext: {},
    search: {},
    limit: 15
};

export default MostCommonVulnerabilities;
