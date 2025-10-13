import React from 'react';
import PropTypes from 'prop-types';
import { Field } from 'redux-form';

const ReduxPasswordField = ({ name, placeholder, disabled }) => (
    <Field
        key={name}
        name={name}
        component="input"
        type="password"
        className={`bg-base-100 border-2 rounded p-2 border-base-300 w-full text-base-600 leading-normal min-h-10 ${
            disabled ? 'bg-base-200' : 'hover:border-base-400'
        }`}
        placeholder={placeholder}
        disabled={disabled}
    />
);

ReduxPasswordField.propTypes = {
    name: PropTypes.string.isRequired,
    placeholder: PropTypes.string,
    disabled: PropTypes.bool,
};

ReduxPasswordField.defaultProps = {
    placeholder: '',
    disabled: false,
};

export default ReduxPasswordField;
