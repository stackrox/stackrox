import React from 'react';
import PropTypes from 'prop-types';
import Widget from 'Components/Widget';
import Sunburst from 'Components/visuals/Sunburst';

import { sunburstData, sunburstRootData, sunburstLegendData } from 'mockData/graphDataMock';

const ComplianceByStandard = ({ type }) => (
    <Widget header={`${type} Compliance`}>
        <Sunburst data={sunburstData} rootData={sunburstRootData} legendData={sunburstLegendData} />
    </Widget>
);

ComplianceByStandard.propTypes = {
    type: PropTypes.string.isRequired,
    params: PropTypes.shape({})
};

ComplianceByStandard.defaultProps = {
    params: null
};

export default ComplianceByStandard;
