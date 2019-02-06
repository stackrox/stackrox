import React from 'react';
import PropTypes from 'prop-types';
import Widget from 'Components/Widget';
import VerticalBarChart from 'Components/visuals/VerticalBar';
import ArcSingle from 'Components/visuals/ArcSingle';
import Query from 'Components/ThrowingQuery';
import Loader from 'Components/Loader';
import pageTypes from 'constants/pageTypes';
import { standardTypes } from 'constants/entityTypes';
import URLService from 'modules/URLService';
import ReactRouterPropTypes from 'react-router-prop-types';
import { withRouter } from 'react-router-dom';
import { resourceLabels } from 'messages/common';
import { AGGREGATED_RESULTS } from 'queries/controls';
import contextTypes from 'constants/contextTypes';
import queryService from 'modules/queryService';
import NoResultsMessage from 'Components/NoResultsMessage';

function getStandardTypeFromName(standardName) {
    if (standardName.includes('NIST')) return standardTypes.NIST_800_190;
    if (standardName.includes('PCI')) return standardTypes.PCI_DSS_3_2;
    if (standardName.includes('HIPAA')) return standardTypes.HIPAA_164;
    if (standardName.includes('CIS_Docker')) return standardTypes.CIS_DOCKER_V1_1_0;
    if (standardName.includes('CIS_Kubernetes')) return standardTypes.CIS_KUBERENETES_V1_2_0;
    return null;
}
const EntityCompliance = ({ entityType, entityName, history }) => {
    const entityTypeLabel = resourceLabels[entityType];

    function getBarData(results) {
        return results.map(item => ({
            x: item.aggregationKeys[0].id,
            y: (item.numPassing / (item.numPassing + item.numFailing)) * 100
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
            entityType: getStandardTypeFromName(datum.x),
            query: {
                [entityType]: entityName
            }
        };
        const URL = URLService.getLinkTo(contextTypes.COMPLIANCE, pageTypes.LIST, linkParams);
        history.push(URL);
    }

    const whereClause = entityName ? { [entityType]: entityName } : null;
    return (
        <Query
            query={AGGREGATED_RESULTS}
            variables={{
                unit: 'CONTROL',
                groupBy: ['STANDARD', entityType],
                where: queryService.objectToWhereClause(whereClause)
            }}
            pollInterval={5000}
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
                                <div className="flex w-full" style={{ alignItems: 'center' }}>
                                    <div className="p-2">
                                        <ArcSingle value={pct} />
                                    </div>
                                    <div className="flex-grow -m-2">
                                        <VerticalBarChart
                                            plotProps={{ height: 180, width: 250 }}
                                            data={barData}
                                            onValueClick={valueClick}
                                        />
                                    </div>
                                </div>
                            </React.Fragment>
                        );
                    }
                }
                return (
                    <Widget header={`${entityTypeLabel} Compliance`} className="sx-2 sy-1">
                        {contents}
                    </Widget>
                );
            }}
        </Query>
    );
};
EntityCompliance.propTypes = {
    entityType: PropTypes.string.isRequired,
    entityName: PropTypes.string,
    history: ReactRouterPropTypes.history.isRequired
};

EntityCompliance.defaultProps = {
    entityName: null
};

export default withRouter(EntityCompliance);
