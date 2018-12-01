import React from 'react';
import PropTypes from 'prop-types';

import * as Icon from 'react-feather';
import { NO_ACCESS, READ_WRITE_ACCESS } from 'constants/accessControl';

const ReadIcon = ({ value, type }) => {
    let compareFunc;
    switch (type) {
        case 'READ':
            compareFunc = value !== NO_ACCESS;
            break;
        case 'WRITE':
            compareFunc = value === READ_WRITE_ACCESS;
            break;
        default:
            compareFunc = false;
    }
    const icon = compareFunc ? (
        <Icon.Check className="text-success-600 h-4 w-4" />
    ) : (
        <Icon.X className="text-alert-600 h-4 w-4" />
    );
    return icon;
};

ReadIcon.propTypes = {
    value: PropTypes.string,
    type: PropTypes.oneOf(['READ', 'WRITE']).isRequired
};

ReadIcon.defaultProps = {
    value: ''
};

export default ReadIcon;
