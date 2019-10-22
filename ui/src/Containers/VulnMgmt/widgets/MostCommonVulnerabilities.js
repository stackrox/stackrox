import React, { useContext } from 'react';
import PropTypes from 'prop-types';
import entityTypes from 'constants/entityTypes';
import { Link } from 'react-router-dom';
import { useQuery } from 'react-apollo';
import gql from 'graphql-tag';
import queryService from 'modules/queryService';
import sortBy from 'lodash/sortBy';

import WorkflowStateMgr from 'modules/WorkflowStateManager';
import workflowStateContext from 'Containers/workflowStateContext';
import { generateURL } from 'modules/URLReadWrite';

import Button from 'Components/Button';
import Loader from 'Components/Loader';
import Widget from 'Components/Widget';
import LabeledBarGraph from 'Components/visuals/LabeledBarGraph';

const MOST_COMMON_VULNERABILITIES = gql`
    query mostCommonVulnerabilities($query: String) {
        results: vulnerabilities(query: $query) {
            id: cve
            cve
            cvss
            scoreVersion
            imageCount
            isFixable
            deploymentCount
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

const getViewAllURL = workflowState => {
    const workflowStateMgr = new WorkflowStateMgr(workflowState);
    workflowStateMgr.pushList(entityTypes.CVE);
    const url = generateURL(workflowStateMgr.workflowState);
    return url;
};

const getSingleEntityURL = (workflowState, id) => {
    const workflowStateMgr = new WorkflowStateMgr(workflowState);
    workflowStateMgr.pushList(entityTypes.CVE).pushListItem(id);
    const url = generateURL(workflowStateMgr.workflowState);
    return url;
};

const processData = (data, workflowState) => {
    const results = sortBy(data.results, [datum => datum.deploymentCount]).splice(-18); // @TODO: Remove when we have pagination on Vulnerabilities
    return results.map(({ id, cve, cvss, scoreVersion, isFixable, deploymentCount }) => {
        const url = getSingleEntityURL(workflowState, id);
        return {
            x: deploymentCount,
            y: `${cve} / CVSS: ${cvss.toFixed(1)} (${scoreVersion}) ${
                isFixable ? ' / Fixable' : ''
            }`,
            url
        };
    });
};

const MostCommonVulnerabilities = ({ entityContext }) => {
    const { loading, data = {} } = useQuery(MOST_COMMON_VULNERABILITIES, {
        variables: {
            query: queryService.entityContextToQueryString(entityContext)
        }
    });

    let content = <Loader />;

    const workflowState = useContext(workflowStateContext);
    if (!loading) {
        const processedData = processData(data, workflowState);

        content = <LabeledBarGraph data={processedData} title="Deployments" />;
    }

    return (
        <Widget
            className="h-full pdf-page"
            bodyClassName="px-2"
            header="Most Common Vulnerabilities"
            headerComponents={<ViewAllButton url={getViewAllURL(workflowState)} />}
        >
            {content}
        </Widget>
    );
};

MostCommonVulnerabilities.propTypes = {
    entityContext: PropTypes.shape({})
};

MostCommonVulnerabilities.defaultProps = {
    entityContext: {}
};

export default MostCommonVulnerabilities;
