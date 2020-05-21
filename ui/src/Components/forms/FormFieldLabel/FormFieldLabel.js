import React from 'react';
import PropTypes from 'prop-types';

const FormFieldLabel = ({ text, required }) => (
    <p>
        {text} {required && <i className="text-base-500">(required)</i>}
    </p>
);

FormFieldLabel.propTypes = {
    text: PropTypes.string.isRequired,
    required: PropTypes.bool,
};

FormFieldLabel.defaultProps = {
    required: false,
};

export default FormFieldLabel;
