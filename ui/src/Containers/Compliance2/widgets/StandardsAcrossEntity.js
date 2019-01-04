import React from 'react';
import PropTypes from 'prop-types';

import Widget from 'Components/Widget';
import HorizontalBarChart from 'Components/visuals/HorizontalBar';

function formatAsPercent(x) {
    return `${x}%`;
}

const StandardsAcrossEntity = ({ type, data }) => (
    <Widget header={`Standards Across ${type}`} className="bg-base-100" bodyClassName="p-2">
        <HorizontalBarChart data={data} valueFormat={formatAsPercent} />
    </Widget>
);

StandardsAcrossEntity.propTypes = {
    type: PropTypes.string.isRequired,
    data: PropTypes.arrayOf(PropTypes.shape()).isRequired
};

export default StandardsAcrossEntity;
