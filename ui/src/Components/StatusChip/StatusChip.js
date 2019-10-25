import React from 'react';
import PropTypes from 'prop-types';

import LabelChip from 'Components/LabelChip';

const StatusChip = ({ status }) => {
    if (!status || !['pass', 'fail'].includes(status)) {
        return '—';
    }

    return status === 'pass' ? (
        <LabelChip text="Pass" type="success" />
    ) : (
        <LabelChip text="Fail" type="alert" />
    );
};

StatusChip.propTypes = {
    status: PropTypes.string
};

StatusChip.defaultProps = {
    status: ''
};

export default StatusChip;
