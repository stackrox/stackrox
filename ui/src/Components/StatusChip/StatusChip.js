import React from 'react';
import PropTypes from 'prop-types';

import LabelChip from 'Components/LabelChip';

const successStates = ['active', 'pass'];
const alertStates = ['inactive', 'fail'];

const StatusChip = ({ status }) => {
    let type = null;
    if (successStates.includes(status)) {
        type = 'success';
    } else if (alertStates.includes(status)) {
        type = 'alert';
    }

    return type ? <LabelChip text={status} type={type} /> : '—';
};

StatusChip.propTypes = {
    status: PropTypes.string
};

StatusChip.defaultProps = {
    status: ''
};

export default StatusChip;
