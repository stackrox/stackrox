import React, { useContext } from 'react';
import PropTypes from 'prop-types';
import Widget from 'Components/Widget';
import VerticalBarChart from 'Components/visuals/VerticalBar';
import ArcSingle from 'Components/visuals/ArcSingle';
import Query from 'Components/ThrowingQuery';
import Loader from 'Components/Loader';
import entityTypes, { standardBaseTypes } from 'constants/entityTypes';
import URLService from 'modules/URLService';
import ReactRouterPropTypes from 'react-router-prop-types';
import { withRouter } from 'react-router-dom';
import { resourceLabels } from 'messages/common';
import { AGGREGATED_RESULTS } from 'queries/controls';
import queryService from 'modules/queryService';
import NoResultsMessage from 'Components/NoResultsMessage';
import { standardLabels } from 'messages/standards';
import searchContext from 'Containers/searchContext';

const EntityCompliance = ({ match, location, entityType, entityName, clusterName, history }) => {
    const entityTypeLabel = resourceLabels[entityType];
    const searchParam = useContext(searchContext);

    function getBarData(results) {
        return results
            .filter(item => item.numPassing + item.numFailing)
            .map(item => ({
                x: standardBaseTypes[item.aggregationKeys[0].id],
                y: (item.numPassing / (item.numPassing + item.numFailing)) * 100,
                standard: item.aggregationKeys[0].id
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
                    standard: standardLabels[datum.standard]
                }
            })
            .url();

        history.push(URL);
    }

    const whereClause = { [entityType]: entityName, [entityTypes.CLUSTER]: clusterName };
    const variables = {
        unit: entityTypes.CHECK,
        groupBy: [entityTypes.STANDARD, entityType],
        where: queryService.objectToWhereClause(whereClause)
    };
    return (
        <Query query={AGGREGATED_RESULTS} variables={variables}>
            {({ loading, data }) => {
                let contents = <Loader />;
                if (!loading && data && data.results) {
                    const { results } = data.results;
                    if (!results.length) {
                        contents = <NoResultsMessage />;
                    } else {
                        const barData = getBarData(results);
                        const totals = getTotals(results);
                        const pct =
                            totals.total > 0
                                ? Math.round((totals.numPassing / totals.total) * 100)
                                : 0;
                        contents = (
                            <React.Fragment>
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
                            </React.Fragment>
                        );
                    }
                }
                return <Widget header={`${entityTypeLabel} Compliance`}>{contents}</Widget>;
            }}
        </Query>
    );
};
EntityCompliance.propTypes = {
    match: ReactRouterPropTypes.match.isRequired,
    location: ReactRouterPropTypes.location.isRequired,
    entityType: PropTypes.string.isRequired,
    entityName: PropTypes.string,
    clusterName: PropTypes.string,
    history: ReactRouterPropTypes.history.isRequired
};

EntityCompliance.defaultProps = {
    entityName: null,
    clusterName: null
};

export default withRouter(EntityCompliance);
