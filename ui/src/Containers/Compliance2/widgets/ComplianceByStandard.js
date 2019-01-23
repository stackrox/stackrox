import React from 'react';
import PropTypes from 'prop-types';

import Widget from 'Components/Widget';
import Sunburst from 'Components/visuals/Sunburst';
import { sunburstData, sunburstLegendData } from 'mockData/graphDataMock';
import Query from 'Components/ThrowingQuery';
import { NODES_QUERY } from 'queries/node';
import { withRouter } from 'react-router-dom';
import Loader from 'Components/Loader';

const ComplianceByStandard = ({ standard, match }) => (
    // TODO: use real query and calculate values based on return data
    <Query query={NODES_QUERY} variables={{ id: match.params.entityId }}>
        {({ loading, data }) => {
            let result = null;
            let contents = <Loader />;

            if (!loading && data) {
                // Temp mock data
                result = sunburstData;
                contents = (
                    <Sunburst data={result} legendData={sunburstLegendData} centerLabel="75%" />
                );
            }

            return <Widget header={`${standard} Compliance`}>{contents}</Widget>;
        }}
    </Query>
);
ComplianceByStandard.propTypes = {
    standard: PropTypes.string.isRequired,
    match: PropTypes.shape({}).isRequired
};

export default withRouter(ComplianceByStandard);
