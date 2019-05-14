import React from 'react';
import PropTypes from 'prop-types';
import { Field } from 'redux-form';

const ReduxPasswordField = ({ name, placeholder }) => (
    <Field
        key={name}
        name={name}
        component="input"
        type="password"
        className="bg-base-100 text-base-600 border rounded w-full p-3 border-base-300"
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
