import React, { useContext } from 'react';
import PropTypes from 'prop-types';
import { useQuery } from 'react-apollo';
import gql from 'graphql-tag';
import sortBy from 'lodash/sortBy';

import workflowStateContext from 'Containers/workflowStateContext';
import ViewAllButton from 'Components/ViewAllButton';
import Loader from 'Components/Loader';
import Widget from 'Components/Widget';
import LabeledBarGraph from 'Components/visuals/LabeledBarGraph';
import NoResultsMessage from 'Components/NoResultsMessage';
import queryService from 'modules/queryService';
import entityTypes from 'constants/entityTypes';
import { cveSortFields } from 'constants/sortFields';
import { WIDGET_PAGINATION_START_OFFSET } from 'constants/workflowPages.constants';
import { getTooltip } from 'utils/vulnerabilityUtils';

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

const processData = (data, workflowState) => {
    // @TODO: filter on the client side until multiple sorts, including derived fields, is supported by BE
    const results = sortBy(data.results, ['cvss']);

    return results.map(vuln => {
        const { id, cve, cvss, scoreVersion, isFixable, deploymentCount } = vuln;
        const url = workflowState.pushRelatedEntity(entityTypes.CVE, id).toUrl();
        const tooltip = getTooltip(vuln);

        return {
            x: deploymentCount,
            y: `${cve} / CVSS: ${cvss.toFixed(1)} (${scoreVersion}) ${
                isFixable ? ' / Fixable' : ''
            }`,
            url,
            hint: tooltip
        };
    });
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
        const processedData = processData(data, workflowState);
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
