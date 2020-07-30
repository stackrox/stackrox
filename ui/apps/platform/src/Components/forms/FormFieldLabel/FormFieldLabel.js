import React from 'react';
import PropTypes from 'prop-types';

import FormFieldRequired from 'Components/forms/FormFieldRequired';

const FormFieldLabel = ({ text, required, empty }) => (
    <p>
        {text} {required && <FormFieldRequired empty={empty} />}
    </p>
);

FormFieldLabel.propTypes = {
    text: PropTypes.string.isRequired,
    required: PropTypes.bool,
    empty: PropTypes.bool,
};

FormFieldLabel.defaultProps = {
    required: false,
    empty: false,
};

export default FormFieldLabel;
