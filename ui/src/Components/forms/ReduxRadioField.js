import React from 'react';
import PropTypes from 'prop-types';
import { Field } from 'redux-form';

const ReduxRadioField = ({ name, value, id, disabled, onChange }) => (
    <Field
        key={name}
        name={name}
        id={id}
        onChange={onChange}
        component="input"
        type="radio"
        className="form-radio"
        value={value}
        disabled={disabled}
    />
);

ReduxRadioField.propTypes = {
    name: PropTypes.string.isRequired,
    value: PropTypes.string.isRequired,
    id: PropTypes.string,
    disabled: PropTypes.bool,
    onChange: PropTypes.func
};

ReduxRadioField.defaultProps = {
    id: null,
    disabled: false,
    onChange: null
};

export default ReduxRadioField;
