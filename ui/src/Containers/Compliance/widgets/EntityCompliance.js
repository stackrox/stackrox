import React from 'react';
import PropTypes from 'prop-types';
import Widget from 'Components/Widget';
import VerticalBarChart from 'Components/visuals/VerticalBar';
import ArcSingle from 'Components/visuals/ArcSingle';
import Query from 'Components/ThrowingQuery';
import Loader from 'Components/Loader';
import pageTypes from 'constants/pageTypes';
import entityTypes, { standardBaseTypes } from 'constants/entityTypes';
import URLService from 'modules/URLService';
import ReactRouterPropTypes from 'react-router-prop-types';
import { withRouter } from 'react-router-dom';
import { resourceLabels } from 'messages/common';
import { AGGREGATED_RESULTS } from 'queries/controls';
import contextTypes from 'constants/contextTypes';
import queryService from 'modules/queryService';
import NoResultsMessage from 'Components/NoResultsMessage';

const EntityCompliance = ({ entityType, entityName, clusterName, history }) => {
    const entityTypeLabel = resourceLabels[entityType];

    function getBarData(results) {
        return results.map(item => ({
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
        const linkParams = {
            entityType: datum.standard,
            query: {
                [entityType]: entityName,
                [entityTypes.CLUSTER]: clusterName
            }
        };
        const URL = URLService.getLinkTo(contextTypes.COMPLIANCE, pageTypes.LIST, linkParams);
        history.push(URL);
    }

    const whereClause = { [entityType]: entityName, [entityTypes.CLUSTER]: clusterName };
    return (
        <Query
            query={AGGREGATED_RESULTS}
            variables={{
                unit: 'CONTROL',
                groupBy: ['STANDARD', entityType],
                where: queryService.objectToWhereClause(whereClause)
            }}
        >
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
                                            plotProps={{ height: 165, width: 300 }}
                                            data={barData}
                                            onValueClick={valueClick}
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
    entityType: PropTypes.string.isRequired,
    entityName: PropTypes.string.isRequired,
    clusterName: PropTypes.string.isRequired,
    history: ReactRouterPropTypes.history.isRequired
};

EntityCompliance.defaultProps = {};

export default withRouter(EntityCompliance);
