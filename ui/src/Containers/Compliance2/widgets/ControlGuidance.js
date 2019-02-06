import { connect } from 'react-redux';
import React from 'react';
import PropTypes from 'prop-types';

import Widget from 'Components/Widget';

const ControlGuidance = ({ interpretationText }) => (
    <Widget header="Control guidance">
        <div className="p-4 leading-loose">{interpretationText}</div>
    </Widget>
);

ControlGuidance.propTypes = {
    interpretationText: PropTypes.string.isRequired
};

export default connect()(ControlGuidance);
