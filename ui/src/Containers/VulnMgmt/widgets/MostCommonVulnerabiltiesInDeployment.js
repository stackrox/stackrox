import React, { useContext } from 'react';
import PropTypes from 'prop-types';
import entityTypes from 'constants/entityTypes';
import { useQuery } from 'react-apollo';
import gql from 'graphql-tag';
import sortBy from 'lodash/sortBy';

import workflowStateContext from 'Containers/workflowStateContext';
import { getVulnerabilityChips } from 'utils/vulnerabilityUtils';

import ViewAllButton from 'Components/ViewAllButton';
import Loader from 'Components/Loader';
import NumberedList from 'Components/NumberedList';
import Widget from 'Components/Widget';

const MOST_COMMON_VULNERABILITIES = gql`
    query mostCommonVulnerabilitiesInDeployment($query: String) {
        results: vulnerabilities(query: $query) {
            id: cve
            cve
            cvss
            scoreVersion
            imageCount
            isFixable
            envImpact
            deployments {
                id
            }
        }
    }
`;

const processData = (data, workflowState, deploymentId, limit) => {
    const results = sortBy(data.results, [datum => datum.imageCount])
        .filter(
            // test whether the given deployment appears in the list of vulnerabilities
            cve =>
                cve.deployments &&
                cve.deployments.some(deployment => deployment.id === deploymentId)
        )
        .splice(-limit); // @TODO: filter on the client side until we have pagination on Vulnerabilities;

    // @TODO: remove JSX generation from processing data and into Numbered List function
    return getVulnerabilityChips(workflowState, results);
};

const MostCommonVulnerabiltiesInDeployment = ({ deploymentId, limit }) => {
    const { loading, data = {} } = useQuery(MOST_COMMON_VULNERABILITIES);

    let content = <Loader />;

    const workflowState = useContext(workflowStateContext);
    if (!loading) {
        const processedData = processData(data, workflowState, deploymentId, limit);

        content = (
            <div className="w-full">
                <NumberedList data={processedData} />
            </div>
        );
    }

    return (
        <Widget
            className="h-full pdf-page"
            bodyClassName="px-2"
            header="Most Common Vulnerabilities"
            headerComponents={
                <ViewAllButton url={workflowState.pushList(entityTypes.CVE).toUrl()} />
            }
        >
            {content}
        </Widget>
    );
};

MostCommonVulnerabiltiesInDeployment.propTypes = {
    deploymentId: PropTypes.string.isRequired,
    limit: PropTypes.number
};

MostCommonVulnerabiltiesInDeployment.defaultProps = {
    limit: 5
};

export default MostCommonVulnerabiltiesInDeployment;
