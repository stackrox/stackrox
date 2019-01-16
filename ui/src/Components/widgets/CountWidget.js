import { connect } from 'react-redux';
import React from 'react';
import PropTypes from 'prop-types';

import Widget from 'Components/Widget';

const CountWidget = ({ title, count, description }) => (
    <Widget
        header={title}
        className="bg-base-100"
        bodyClassName="flex-col h-full justify-center text-center"
    >
        <div className="text-6xl font-500">{count}</div>
        <div className="text-base-500 pt-1">{description}</div>
    </Widget>
);

CountWidget.propTypes = {
    title: PropTypes.string.isRequired,
    count: PropTypes.string.isRequired,
    description: PropTypes.string
};

CountWidget.defaultProps = {
    description: null
};

export default connect()(CountWidget);
