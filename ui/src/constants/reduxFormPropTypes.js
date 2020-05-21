import PropTypes from 'prop-types';

export default {
    fields: PropTypes.shape({
        push: PropTypes.func,
        length: PropTypes.number,
        remove: PropTypes.func,
        map: PropTypes.func,
        get: PropTypes.func,
    }).isRequired,
};
