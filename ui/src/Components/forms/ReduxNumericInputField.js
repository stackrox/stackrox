import React from 'react';
import PropTypes from 'prop-types';
import { Field } from 'redux-form';
import NumericInput from 'react-numeric-input';

const ReduxNumericInput = props => (
    <NumericInput
        max={props.max}
        min={props.min}
        key={props.input.value}
        field={props.input.value}
        id={props.input.value}
        value={props.input.value}
        placeholder={props.placeholder}
        onBlur={props.input.onChange}
        style={{ style: false }}
        className="border rounded-l p-3 border-base-300 w-full font-400"
    />
);

ReduxNumericInput.propTypes = {
    input: PropTypes.shape({
        value: PropTypes.oneOfType([PropTypes.number, PropTypes.string]),
        onChange: PropTypes.func
    }).isRequired,
    placeholder: PropTypes.string.isRequired,
    min: PropTypes.number.isRequired,
    max: PropTypes.number.isRequired
};

const ReduxNumericInputField = ({ name, min, max, placeholder }) => (
    <Field
        key={name}
        name={name}
        placeholder={placeholder}
        parse={parseInt}
        min={min}
        max={max}
        component={ReduxNumericInput}
        className="border bg-white border-base-300 text-base-600 p-3 pr-8 rounded-r-sm cursor-pointer z-1 focus:border-base-300 w-full font-400"
    />
);

ReduxNumericInputField.propTypes = {
    name: PropTypes.string.isRequired,
    min: PropTypes.number.isRequired,
    max: PropTypes.number.isRequired,
    placeholder: PropTypes.string.isRequired
};

export default ReduxNumericInputField;
