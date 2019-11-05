import React from 'react';
import PropTypes from 'prop-types';

import LabelChip from 'Components/LabelChip';

const successStates = ['active', 'pass'];
const alertStates = ['inactive', 'fail'];

const StatusChip = ({ status, size }) => {
    let type = null;
    if (successStates.includes(status)) {
        type = 'success';
    } else if (alertStates.includes(status)) {
        type = 'alert';
    }

    return type ? <LabelChip text={status} type={type} size={size} /> : '—';
};

StatusChip.propTypes = {
    status: PropTypes.string,
    size: PropTypes.string
};

StatusChip.defaultProps = {
    status: '',
    size: 'large'
};

export default StatusChip;
