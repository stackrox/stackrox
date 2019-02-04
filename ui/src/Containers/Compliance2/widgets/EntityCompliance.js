import React from 'react';
import PropTypes from 'prop-types';
import Widget from 'Components/Widget';
import VerticalBarChart from 'Components/visuals/VerticalBar';
import ArcSingle from 'Components/visuals/ArcSingle';
import Query from 'Components/AppQuery';
import componentTypes from 'constants/componentTypes';
import Loader from 'Components/Loader';
import pageTypes from 'constants/pageTypes';
import { standardTypes } from 'constants/entityTypes';
import URLService from 'modules/URLService';
import ReactRouterPropTypes from 'react-router-prop-types';
import { withRouter } from 'react-router-dom';
import resourceLabels from 'messages/common';

function getStandardTypeFromName(standardName) {
    if (standardName.includes('NIST')) return standardTypes.NIST_800_190;
    if (standardName.includes('PCI')) return standardTypes.PCI_DSS_3_2;
    if (standardName.includes('HIPAA')) return standardTypes.HIPAA_164;
    if (standardName.includes('CIS Docker')) return standardTypes.CIS_DOCKER_V1_1_0;
    if (standardName.includes('CIS Kubernetes')) return standardTypes.CIS_KUBERENETES_V1_2_0;
    return null;
}
const EntityCompliance = ({ params, history }) => {
    const { entityType } = params;
    const entityTypeLabel = resourceLabels[entityType];

    function getBarData(results) {
        return results.aggregatedResults.results.map(item => ({
            x: item.aggregationKeys[0].id,
            y: (item.numPassing / (item.numPassing + item.numFailing)) * 100
        }));
    }

    function getTotals(results) {
        return results.aggregatedResults.results.reduce(
            (acc, curr) => {
                acc.numPassing += curr.numPassing;
                acc.total += curr.numPassing + curr.numFailing;
                return acc;
            },
            { numPassing: 0, total: 0 }
        );
    }
    function valueClick(datum) {
        const { context } = params;
        const linkParams = {
            entityType: getStandardTypeFromName(datum.x),
            query: {
                [entityType]: params.entityId
            }
        };

        const URL = URLService.getLinkTo(context, pageTypes.LIST, linkParams);
        history.push(URL);
    }

    return (
        <Query params={params} componentType={componentTypes.ENTITY_COMPLIANCE}>
            {({ loading, data }) => {
                let contents = <Loader />;
                if (!loading && data && data.results) {
                    const barData = getBarData(data.results);
                    const totals = getTotals(data.results);
                    const pct = Math.round((totals.numPassing / totals.total) * 100);
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
    params: PropTypes.shape({
        entityType: PropTypes.string
    }).isRequired,
    history: ReactRouterPropTypes.history.isRequired
};

export default withRouter(EntityCompliance);
