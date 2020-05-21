import PropTypes from 'prop-types';

const severityPropType = PropTypes.oneOf([
    'CRITICAL_SEVERITY',
    'HIGH_SEVERITY',
    'MEDIUM_SEVERITY',
    'LOW_SEVERITY',
]);

export default severityPropType;
