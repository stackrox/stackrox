import React, { useContext } from 'react';
import PropTypes from 'prop-types';
import { gql, useQuery } from '@apollo/client';
import sortBy from 'lodash/sortBy';

import entityTypes from 'constants/entityTypes';
import queryService from 'utils/queryService';
import workflowStateContext from 'Containers/workflowStateContext';
import { getVulnerabilityChips } from 'utils/vulnerabilityUtils';
import { cveSortFields } from 'constants/sortFields';
import { WIDGET_PAGINATION_START_OFFSET } from 'constants/workflowPages.constants';
import Loader from 'Components/Loader';
import NumberedList from 'Components/NumberedList';
import Widget from 'Components/Widget';
import NoResultsMessage from 'Components/NoResultsMessage';

import ViewAllButton from './ViewAllButton';

const MOST_COMMON_IMAGE_VULNERABILITIES = gql`
    query mostCommonImageVulnerabilities($query: String, $vulnPagination: Pagination) {
        results: imageVulnerabilities(query: $query, pagination: $vulnPagination) {
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
    const results = sortBy(data.results, ['cvss']);

    // @TODO: remove JSX generation from processing data and into Numbered List function
    return getVulnerabilityChips(workflowState, results);
};

const MostCommonVulnerabiltiesInDeployment = ({ deploymentId, limit }) => {
    const { loading, data = {} } = useQuery(MOST_COMMON_IMAGE_VULNERABILITIES, {
        variables: {
            query: queryService.objectToWhereClause({
                'Deployment ID': deploymentId,
            }),
            scopeQuery: queryService.objectToWhereClause({
                'Deployment ID': deploymentId,
            }),
            vulnPagination: queryService.getPagination(
                {
                    id: cveSortFields.IMAGE_COUNT,
                    desc: true,
                },
                WIDGET_PAGINATION_START_OFFSET,
                limit
            ),
        },
    });

    let content = <Loader />;

    const workflowState = useContext(workflowStateContext);
    if (!loading) {
        const processedData = processData(data, workflowState);

        if (!processedData || processedData.length === 0) {
            content = (
                <NoResultsMessage message="No vulnerabilities found" className="p-6" icon="info" />
            );
        } else {
            content = (
                <div className="w-full">
                    <NumberedList data={processedData} />
                </div>
            );
        }
    }

    const viewAllURL = workflowState
        .pushList(entityTypes.CVE)
        .setSort([
            { id: cveSortFields.IMAGE_COUNT, desc: true },
            { id: cveSortFields.CVSS_SCORE, desc: true },
        ])
        .toUrl();

    return (
        <Widget
            className="h-full pdf-page"
            header="Most common image vulnerabilities"
            headerComponents={<ViewAllButton url={viewAllURL} />}
        >
            {content}
        </Widget>
    );
};

MostCommonVulnerabiltiesInDeployment.propTypes = {
    deploymentId: PropTypes.string.isRequired,
    limit: PropTypes.number,
};

MostCommonVulnerabiltiesInDeployment.defaultProps = {
    limit: 5,
};

export default MostCommonVulnerabiltiesInDeployment;
