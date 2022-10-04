import React, { useContext } from 'react';
import PropTypes from 'prop-types';
import { gql, useQuery } from '@apollo/client';
import sortBy from 'lodash/sortBy';

import entityTypes from 'constants/entityTypes';
import queryService from 'utils/queryService';
import workflowStateContext from 'Containers/workflowStateContext';
import ViewAllButton from 'Components/ViewAllButton';
import Loader from 'Components/Loader';
import Widget from 'Components/Widget';
import NumberedList from 'Components/NumberedList';
import { checkForPermissionErrorMessage } from 'utils/permissionUtils';
import { getVulnerabilityChips } from 'utils/vulnerabilityUtils';
import NoResultsMessage from 'Components/NoResultsMessage';
import { cveSortFields } from 'constants/sortFields';
import useFeatureFlags from 'hooks/useFeatureFlags';

// TODO: remove once ROX_POSTGRES_DATASTORE is enabled
export const RECENTLY_DETECTED_VULNERABILITIES = gql`
    query recentlyDetectedVulnerabilities(
        $query: String
        $scopeQuery: String
        $pagination: Pagination
    ) {
        results: vulnerabilities(query: $query, pagination: $pagination) {
            id
            cve
            cvss
            scoreVersion
            deploymentCount
            imageCount
            isFixable(query: $scopeQuery)
            envImpact
            createdAt
            summary
        }
    }
`;

export const RECENTLY_DETECTED_IMAGE_VULNERABILITIES = gql`
    query recentlyDetectedImageVulnerabilities(
        $query: String
        $scopeQuery: String
        $pagination: Pagination
    ) {
        results: imageVulnerabilities(query: $query, pagination: $pagination) {
            id
            cve
            cvss
            scoreVersion
            deploymentCount
            imageCount
            isFixable(query: $scopeQuery)
            envImpact
            createdAt
            summary
        }
    }
`;

const processData = (data, workflowState, cveType) => {
    let results = data && data.results && data.results.filter((datum) => datum.createdAt);
    // @TODO: filter on the client side until multiple sorts, including derived fields, is supported by BE
    results = sortBy(results, ['createdAt', 'cvss', 'envImpact']).reverse();

    // @TODO: remove JSX generation from processing data and into Numbered List function
    return getVulnerabilityChips(workflowState, results, cveType);
};

const RecentlyDetectedImageVulnerabilities = ({ entityContext, search, limit }) => {
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const showVmUpdates = isFeatureFlagEnabled('ROX_POSTGRES_DATASTORE');
    const entityContextObject = queryService.entityContextToQueryObject(entityContext); // deals with BE inconsistency

    const queryObject = {
        ...entityContextObject,
        ...search,
        [cveSortFields.CVE_TYPE]: 'IMAGE_CVE',
    }; // Combine entity context and search
    const query = queryService.objectToWhereClause(queryObject); // get final gql query string

    const {
        loading,
        data = {},
        error,
    } = useQuery(
        showVmUpdates ? RECENTLY_DETECTED_IMAGE_VULNERABILITIES : RECENTLY_DETECTED_VULNERABILITIES,
        {
            variables: {
                query,
                scopeQuery: queryService.objectToWhereClause(entityContextObject),
                pagination: queryService.getPagination(
                    {
                        id: cveSortFields.CVE_CREATED_TIME,
                        desc: true,
                    },
                    0,
                    limit
                ),
            },
        }
    );

    let content = <Loader />;

    const workflowState = useContext(workflowStateContext);
    if (!loading) {
        if (error) {
            const defaultMessage = `An error occurred in retrieving vulnerabilities. Please refresh the page. If this problem continues, please contact support.`;

            const parsedMessage = checkForPermissionErrorMessage(error, defaultMessage);

            content = <NoResultsMessage message={parsedMessage} className="p-3" icon="warn" />;
        } else {
            const processedData = processData(
                data,
                workflowState,
                showVmUpdates ? entityTypes.IMAGE_CVE : entityTypes.CVE
            );

            if (!processedData || processedData.length === 0) {
                content = (
                    <NoResultsMessage
                        message="No vulnerabilities found"
                        className="p-3"
                        icon="info"
                    />
                );
            } else {
                content = (
                    <div className="w-full">
                        <NumberedList data={processedData} />
                    </div>
                );
            }
        }
    }

    const viewAllURL = workflowState
        .pushList(showVmUpdates ? entityTypes.IMAGE_CVE : entityTypes.CVE)
        .setSort([{ id: cveSortFields.CVE_CREATED_TIME, desc: true }])
        .toUrl();

    return (
        <Widget
            className="h-full pdf-page"
            header={`Recently Detected ${showVmUpdates ? 'Image ' : ''}Vulnerabilities`}
            headerComponents={<ViewAllButton url={viewAllURL} />}
        >
            {content}
        </Widget>
    );
};

RecentlyDetectedImageVulnerabilities.propTypes = {
    entityContext: PropTypes.shape({}),
    search: PropTypes.shape({}),
    limit: PropTypes.number,
};

RecentlyDetectedImageVulnerabilities.defaultProps = {
    entityContext: {},
    search: {},
    limit: 5,
};

export default RecentlyDetectedImageVulnerabilities;
