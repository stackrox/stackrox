import React from 'react';
import PropTypes from 'prop-types';

const FormFieldRequired = ({ empty }) => (
    <i className={empty ? 'text-warning-500' : 'text-base-500'}>(required)</i>
);

FormFieldRequired.propTypes = {
    empty: PropTypes.bool.isRequired,
};

export default FormFieldRequired;
