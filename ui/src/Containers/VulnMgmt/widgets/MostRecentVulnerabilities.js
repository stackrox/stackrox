import React, { useContext } from 'react';
import PropTypes from 'prop-types';
import entityTypes from 'constants/entityTypes';
import { Link } from 'react-router-dom';
import { useQuery } from 'react-apollo';
import gql from 'graphql-tag';
import queryService from 'modules/queryService';
import pluralize from 'pluralize';
import sortBy from 'lodash/sortBy';
import { getTime } from 'date-fns';

import WorkflowStateMgr from 'modules/WorkflowStateManager';
import workflowStateContext from 'Containers/workflowStateContext';
import { generateURL } from 'modules/URLReadWrite';

import Button from 'Components/Button';
import Loader from 'Components/Loader';
import Widget from 'Components/Widget';
import NumberedList from 'Components/NumberedList';
import LabelChip from 'Components/LabelChip';

const MOST_RECENT_VULNERABILITIES = gql`
    query mostRecentVulnerabilities($query: String) {
        results: vulnerabilities(query: $query) {
            id: cve
            cve
            cvss
            scoreVersion
            imageCount
            isFixable
            envImpact
            lastScanned
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
    const results = sortBy(data.results, [datum => getTime(new Date(datum.lastScanned))])
        .splice(-8)
        .reverse(); // @TODO: Remove when we have pagination on Vulnerabilities
    return results.map(({ id, cve, cvss, scoreVersion, imageCount, envImpact, isFixable }) => {
        const text = cve;
        const envImpactPercentage = `${(envImpact * 100).toFixed(0)}%`;
        return {
            text,
            subText: `/ CVSS: ${cvss.toFixed(1)} (${scoreVersion})`,
            url: getSingleEntityURL(workflowState, id),
            component: (
                <>
                    {imageCount > 0 && (
                        <div className="ml-2">
                            <LabelChip
                                text={`${imageCount} ${pluralize('Image', imageCount)}`}
                                type="tertiary"
                                size="small"
                            />
                        </div>
                    )}
                    {envImpact && (
                        <div className="ml-2">
                            <LabelChip
                                text={`Env Impact: ${envImpactPercentage}`}
                                type="secondary"
                                size="small"
                            />
                        </div>
                    )}
                    {isFixable && (
                        <div className="ml-2">
                            <LabelChip text="Fixable" type="success" size="small" />
                        </div>
                    )}
                </>
            )
        };
    });
};

const MostRecentVulnerabilities = ({ entityContext }) => {
    const { loading, data = {} } = useQuery(MOST_RECENT_VULNERABILITIES, {
        variables: {
            query: queryService.entityContextToQueryString(entityContext)
        }
    });

    let content = <Loader />;

    const workflowState = useContext(workflowStateContext);
    if (!loading) {
        const processedData = processData(data, workflowState);

        content = (
            <div className="w-full">
                <NumberedList data={processedData} />
            </div>
        );
    }

    return (
        <Widget
            className="s-2 pdf-page"
            header="Most Recent Vulnerabilities"
            headerComponents={<ViewAllButton url={getViewAllURL(workflowState)} />}
        >
            {content}
        </Widget>
    );
};

MostRecentVulnerabilities.propTypes = {
    entityContext: PropTypes.shape({})
};

MostRecentVulnerabilities.defaultProps = {
    entityContext: {}
};

export default MostRecentVulnerabilities;
