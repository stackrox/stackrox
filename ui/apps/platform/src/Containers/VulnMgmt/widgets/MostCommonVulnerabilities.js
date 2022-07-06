import React, { useContext } from 'react';
import PropTypes from 'prop-types';
import { gql, useQuery } from '@apollo/client';
import sortBy from 'lodash/sortBy';

import workflowStateContext from 'Containers/workflowStateContext';
import ViewAllButton from 'Components/ViewAllButton';
import Loader from 'Components/Loader';
import Widget from 'Components/Widget';
import LabeledBarGraph from 'Components/visuals/LabeledBarGraph';
import NoResultsMessage from 'Components/NoResultsMessage';
import { checkForPermissionErrorMessage } from 'utils/permissionUtils';
import queryService from 'utils/queryService';
import entityTypes from 'constants/entityTypes';
import { cveSortFields } from 'constants/sortFields';
import { WIDGET_PAGINATION_START_OFFSET } from 'constants/workflowPages.constants';
import { getTooltip } from 'utils/vulnerabilityUtils';
import useFeatureFlags from 'hooks/useFeatureFlags';

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

const processData = (data, workflowState, showVMUpdates) => {
    // @TODO: filter on the client side until multiple sorts, including derived fields, is supported by BE
    const results = sortBy(data.results, ['cvss']);

    return results.map((vuln) => {
        const { id, cve, cvss, scoreVersion, isFixable, deploymentCount, imageCount } = vuln;
        const url = workflowState.pushRelatedEntity(entityTypes.CVE, id).toUrl();
        const tooltip = getTooltip(vuln);
        const count = showVMUpdates ? imageCount : deploymentCount;

        return {
            x: count,
            y: `${cve} / CVSS: ${cvss.toFixed(1)} (${scoreVersion}) ${
                isFixable ? ' / Fixable' : ''
            }`,
            url,
            hint: tooltip,
        };
    });
};

const MostCommonVulnerabilities = ({ entityContext, search, limit }) => {
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const showVMUpdates = isFeatureFlagEnabled('ROX_FRONTEND_VM_UDPATES');

    const entityContextObject = queryService.entityContextToQueryObject(entityContext); // deals with BE inconsistency

    const queryObject = { ...entityContextObject, ...search }; // Combine entity context and search
    const query = queryService.objectToWhereClause(queryObject); // get final gql query string
    const GQL_QUERY = showVMUpdates
        ? MOST_COMMON_IMAGE_VULNERABILITIES
        : MOST_COMMON_VULNERABILITIES;
    const sortFieldID = showVMUpdates ? cveSortFields.IMAGE_COUNT : cveSortFields.DEPLOYMENT_COUNT;
    const labeledBarGraphTitle = showVMUpdates ? 'Images' : 'Deployments';

    const {
        loading,
        data = {},
        error,
    } = useQuery(GQL_QUERY, {
        variables: {
            query,
            vulnPagination: queryService.getPagination(
                {
                    id: sortFieldID,
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
        if (error) {
            const defaultMessage = `An error occurred in retrieving vulnerabilities. Please refresh the page. If this problem continues, please contact support.`;

            const parsedMessage = checkForPermissionErrorMessage(error, defaultMessage);

            content = <NoResultsMessage message={parsedMessage} className="p-3" icon="warn" />;
        } else {
            const processedData = processData(data, workflowState, showVMUpdates);
            if (!processedData || processedData.length === 0) {
                content = (
                    <NoResultsMessage
                        message="No vulnerabilities found"
                        className="p-3"
                        icon="info"
                    />
                );
            } else {
                content = <LabeledBarGraph data={processedData} title={labeledBarGraphTitle} />;
            }
        }
    }

    const header = showVMUpdates
        ? 'Most Common Image Vulnerabilities'
        : 'Most Common Vulnerabilities';
    const viewAllURL = workflowState
        .pushList(entityTypes.CVE)
        .setSort([
            { id: sortFieldID, desc: true },
            { id: cveSortFields.CVSS_SCORE, desc: true },
        ])
        .toUrl();

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

MostCommonVulnerabilities.propTypes = {
    entityContext: PropTypes.shape({}),
    search: PropTypes.shape({}),
    limit: PropTypes.number,
};

MostCommonVulnerabilities.defaultProps = {
    entityContext: {},
    search: {},
    limit: 15,
};

export default MostCommonVulnerabilities;
