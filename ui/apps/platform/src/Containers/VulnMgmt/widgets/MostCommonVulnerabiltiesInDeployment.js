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
import ViewAllButton from 'Components/ViewAllButton';
import Loader from 'Components/Loader';
import NumberedList from 'Components/NumberedList';
import Widget from 'Components/Widget';
import NoResultsMessage from 'Components/NoResultsMessage';
import useFeatureFlags from 'hooks/useFeatureFlags';

const MOST_COMMON_VULNERABILITIES = gql`
    query mostCommonVulnerabilitiesInDeployment(
        $query: String
        $scopeQuery: String
        $vulnPagination: Pagination
    ) {
        results: vulnerabilities(query: $query, pagination: $vulnPagination) {
            id
            cve
            cvss
            scoreVersion
            imageCount
            deploymentCount
            createdAt
            summary
            isFixable(query: $scopeQuery)
            envImpact
        }
    }
`;

const MOST_COMMON_IMAGE_VULNERABILITIES = gql`
    query mostCommonImageVulnerabilities($query: String, $vulnPagination: Pagination) {
        results: imageVulnerabilities(query: $query, pagination: $vulnPagination) {
            id
            cve
            cvss
            scoreVersion
            isFixable
            deploymentCount
            imageCount
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
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const showVMUpdates = isFeatureFlagEnabled('ROX_FRONTEND_VM_UPDATES');

    const queryToUse = showVMUpdates
        ? MOST_COMMON_IMAGE_VULNERABILITIES
        : MOST_COMMON_VULNERABILITIES;

    const { loading, data = {} } = useQuery(queryToUse, {
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

    const header = showVMUpdates
        ? 'Most Common Image Vulnerabilities'
        : 'Most Common Vulnerabilities';

    return (
        <Widget
            className="h-full pdf-page"
            header={header}
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
