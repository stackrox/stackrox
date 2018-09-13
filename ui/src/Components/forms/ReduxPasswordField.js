import React from 'react';
import PropTypes from 'prop-types';
import { Field } from 'redux-form';

const ReduxPasswordField = ({ name, placeholder }) => (
    <Field
        key={name}
        name={name}
        component="input"
        type="password"
        className="border rounded w-full p-3 border-base-300"
        placeholder={placeholder}
    />
);

ReduxPasswordField.propTypes = {
    name: PropTypes.string.isRequired,
    placeholder: PropTypes.string
};

ReduxPasswordField.defaultProps = {
    placeholder: ''
};

export default ReduxPasswordField;
