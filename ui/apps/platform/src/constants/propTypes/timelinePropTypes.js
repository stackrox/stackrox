import PropTypes from 'prop-types';

export const eventPropTypes = {
    name: PropTypes.string.isRequired,
    args: PropTypes.string,
    type: PropTypes.string.isRequired,
    uid: PropTypes.number,
    parentName: PropTypes.string,
    parentUid: PropTypes.number,
    reason: PropTypes.string,
    timestamp: PropTypes.string.isRequired,
    inBaseline: PropTypes.bool,
};

export const defaultEventPropTypes = {
    args: null,
    uid: null,
    parentName: null,
    parentUid: null,
    reason: null,
    inBaseline: null,
};

export const clusteredEventPropTypes = {
    size: PropTypes.number.isRequired,
    numEvents: PropTypes.number.isRequired,
};
