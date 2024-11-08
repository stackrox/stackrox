import React, { useContext } from 'react';
import PropTypes from 'prop-types';
import { useRouteMatch, useLocation, useHistory } from 'react-router-dom';

import Widget from 'Components/Widget';
import ArcSingle from 'Components/visuals/ArcSingle';
import Query from 'Components/CacheFirstQuery';
import Loader from 'Components/Loader';
import entityTypes, { standardBaseTypes } from 'constants/entityTypes';
import URLService from 'utils/URLService';
import { AGGREGATED_RESULTS } from 'queries/controls';
import queryService from 'utils/queryService';
import NoResultsMessage from 'Components/NoResultsMessage';
import { standardLabels } from 'messages/standards';
import searchContext from 'Containers/searchContext';

import { entityNounSentenceCaseSingular } from '../entitiesForCompliance';
import VerticalBarChart from './VerticalBarChart';

const EntityCompliance = ({ entityType, entityName, clusterName }) => {
    const entityTypeLabel = entityNounSentenceCaseSingular[entityType];
    const searchParam = useContext(searchContext);
    const match = useRouteMatch();
    const location = useLocation();
    const history = useHistory();

    function getBarData(results) {
        return results
            .filter((item) => item.numPassing + item.numFailing)
            .map((item) => ({
                x: standardBaseTypes[item.aggregationKeys[0].id] || item.aggregationKeys[0].id,
                y: (item.numPassing / (item.numPassing + item.numFailing)) * 100,
                standard: item.aggregationKeys[0].id,
            }));
    }

    function getTotals(results) {
        return results.reduce(
            (acc, curr) => {
                acc.numPassing += curr.numPassing;
                acc.total += curr.numPassing + curr.numFailing;
                return acc;
            },
            { numPassing: 0, total: 0 }
        );
    }
    function valueClick(datum) {
        const URL = URLService.getURL(match, location)
            .base(entityTypes.CONTROL)
            .query({
                [searchParam]: {
                    [entityType]: entityName,
                    [entityTypes.CLUSTER]: clusterName,
                    standard: standardLabels[datum.standard] || datum.x,
                },
            })
            .url();

        history.push(URL);
    }

    const whereClause = { [entityType]: entityName, [entityTypes.CLUSTER]: clusterName };
    const variables = {
        unit: entityTypes.CHECK,
        groupBy: [entityTypes.STANDARD, entityType],
        where: queryService.objectToWhereClause(whereClause),
    };
    return (
        <Query query={AGGREGATED_RESULTS} variables={variables}>
            {({ loading, data }) => {
                let contents = <Loader />;
                if (!loading && data && data.results) {
                    // Frontend filtering of results.
                    const { complianceStandards } = data;
                    const results = data.results.results.filter((result) => {
                        const standardId = result.aggregationKeys[0].id;
                        return complianceStandards.some(({ id }) => id === standardId);
                    });

                    if (!results.length) {
                        contents = (
                            <NoResultsMessage message="No data available. Please ensure your cluster is properly configured." />
                        );
                    } else {
                        const barData = getBarData(results);
                        const totals = getTotals(results);
                        const pct =
                            totals.total > 0
                                ? Math.round((totals.numPassing / totals.total) * 100)
                                : 0;
                        contents = (
                            <>
                                <div className="flex w-full items-center">
                                    <div className="px-2">
                                        <ArcSingle value={pct} />
                                    </div>
                                    <div
                                        className="w-full flex justify-end overflow-hidden relative"
                                        style={{ maxHeight: '129px' }}
                                    >
                                        <VerticalBarChart
                                            plotProps={{ height: 145 }}
                                            data={barData}
                                            onValueClick={valueClick}
                                            legend={false}
                                        />
                                    </div>
                                </div>
                            </>
                        );
                    }
                }
                return <Widget header={`${entityTypeLabel} Compliance`}>{contents}</Widget>;
            }}
        </Query>
    );
};
EntityCompliance.propTypes = {
    entityType: PropTypes.string.isRequired,
    entityName: PropTypes.string,
    clusterName: PropTypes.string,
};

EntityCompliance.defaultProps = {
    entityName: null,
    clusterName: null,
};

export default EntityCompliance;
