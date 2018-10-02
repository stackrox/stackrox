import React from 'react';
import PropTypes from 'prop-types';
import { Field } from 'redux-form';

const ReduxTextField = ({ name, disabled, placeholder }) => (
    <Field
        key={name}
        name={name}
        component="input"
        type="text"
        className={`border rounded-l p-3 border-base-300 w-full font-400 ${
            disabled ? 'bg-base-200' : ''
        }`}
        disabled={disabled}
        autoComplete=""
        placeholder={placeholder}
    />
);

ReduxTextField.propTypes = {
    name: PropTypes.string.isRequired,
    disabled: PropTypes.bool,
    placeholder: PropTypes.string
};

ReduxTextField.defaultProps = {
    disabled: false,
    placeholder: ''
};

export default ReduxTextField;
