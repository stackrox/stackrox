import React, { useContext } from 'react';
import PropTypes from 'prop-types';
import entityTypes from 'constants/entityTypes';
import { useQuery } from 'react-apollo';
import gql from 'graphql-tag';
import queryService from 'modules/queryService';
import sortBy from 'lodash/sortBy';
import { getTime } from 'date-fns';

import workflowStateContext from 'Containers/workflowStateContext';

import ViewAllButton from 'Components/ViewAllButton';
import Loader from 'Components/Loader';
import Widget from 'Components/Widget';
import NumberedList from 'Components/NumberedList';
import { getVulnerabilityChips, parseCVESearch } from 'utils/vulnerabilityUtils';

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

const processData = (data, workflowState, limit) => {
    const results = sortBy(data.results, [datum => getTime(new Date(datum.lastScanned))])
        .splice(-limit) // @TODO: filter on the client side until we have pagination on Vulnerabilities
        .reverse(); // @TODO: Remove when we have pagination on Vulnerabilities

    // @TODO: remove JSX generation from processing data and into Numbered List function
    return getVulnerabilityChips(workflowState, results);
};

const MostRecentVulnerabilities = ({ entityContext, search, limit }) => {
    const entityContextObject = queryService.entityContextToQueryObject(entityContext); // deals with BE inconsistency

    const parsedSearch = parseCVESearch(search); // hack until isFixable is allowed in search

    const queryObject = { ...entityContextObject, ...parsedSearch }; // Combine entity context and search
    const query = queryService.objectToWhereClause(queryObject); // get final gql query string

    const { loading, data = {} } = useQuery(MOST_RECENT_VULNERABILITIES, {
        variables: {
            query
        }
    });

    let content = <Loader />;

    const workflowState = useContext(workflowStateContext);
    if (!loading) {
        const processedData = processData(data, workflowState, limit);

        content = (
            <div className="w-full">
                <NumberedList data={processedData} />
            </div>
        );
    }

    return (
        <Widget
            className="h-full pdf-page"
            header="Most Recent Vulnerabilities"
            headerComponents={
                <ViewAllButton url={workflowState.pushList(entityTypes.CVE).toUrl()} />
            }
        >
            {content}
        </Widget>
    );
};

MostRecentVulnerabilities.propTypes = {
    entityContext: PropTypes.shape({}),
    search: PropTypes.shape({}),
    limit: PropTypes.number
};

MostRecentVulnerabilities.defaultProps = {
    entityContext: {},
    search: {},
    limit: 5
};

export default MostRecentVulnerabilities;
