import React from 'react';
import PropTypes from 'prop-types';
import { Field } from 'redux-form';

const ReduxCheckboxField = ({ name, id, disabled, onChange }) => (
    <Field
        key={name}
        name={name}
        id={id}
        onChange={onChange}
        component="input"
        type="checkbox"
        disabled={disabled}
    />
);

ReduxCheckboxField.propTypes = {
    name: PropTypes.string.isRequired,
    id: PropTypes.string,
    disabled: PropTypes.bool,
    onChange: PropTypes.func
};

ReduxCheckboxField.defaultProps = {
    id: null,
    disabled: false,
    onChange: null
};

export default ReduxCheckboxField;
