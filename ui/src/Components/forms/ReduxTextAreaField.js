import React from 'react';
import PropTypes from 'prop-types';
import { Field } from 'redux-form';

const ReduxTextAreaField = ({ name, disabled, placeholder, maxlength }) => (
    <Field
        key={name}
        name={name}
        component="textarea"
        className="border rounded-l py-3 px-2 border-base-300 text-base-600 w-full font-600 leading-normal"
        disabled={disabled}
        rows={4}
        placeholder={placeholder}
        maxlength={maxlength}
    />
);

ReduxTextAreaField.propTypes = {
    name: PropTypes.string.isRequired,
    disabled: PropTypes.bool,
    placeholder: PropTypes.string.isRequired,
    maxlength: PropTypes.string
};

ReduxTextAreaField.defaultProps = {
    disabled: false,
    maxlength: null
};

export default ReduxTextAreaField;
