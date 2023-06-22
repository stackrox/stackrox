import React from 'react';
import PropTypes from 'prop-types';
import { Field } from 'redux-form';

const ReduxTextField = ({ name, disabled, placeholder }) => (
    <Field
        key={name}
        name={name}
        component="input"
        type="text"
        className={`bg-base-100 border-2 rounded p-2 border-base-300 w-full text-base-600 leading-normal min-h-10 ${
            disabled ? 'bg-base-200' : 'hover:border-base-400'
        }`}
        disabled={disabled}
        autoComplete=""
        placeholder={placeholder}
    />
);

ReduxTextField.propTypes = {
    name: PropTypes.string.isRequired,
    disabled: PropTypes.bool,
    placeholder: PropTypes.string,
};

ReduxTextField.defaultProps = {
    disabled: false,
    placeholder: '',
};

export default ReduxTextField;
