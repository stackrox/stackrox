import React from 'react';
import PropTypes from 'prop-types';
import { Field } from 'redux-form';
import NumericInput from 'react-numeric-input';

const ReduxNumericInput = props => (
    <NumericInput
        max={props.max}
        min={props.min}
        step={props.step}
        key={props.input.value}
        field={props.input.value}
        id={props.input.value}
        value={props.input.value}
        placeholder={props.placeholder}
        onBlur={props.input.onChange}
        noStyle
        className="bg-base-100 border-2 rounded-l p-3 text-base-600 border-base-300 w-full font-600"
    />
);

ReduxNumericInput.propTypes = {
    input: PropTypes.shape({
        value: PropTypes.oneOfType([PropTypes.number, PropTypes.string]),
        onChange: PropTypes.func
    }).isRequired,
    placeholder: PropTypes.string.isRequired,
    min: PropTypes.number.isRequired,
    max: PropTypes.number.isRequired,
    step: PropTypes.number.isRequired
};

const ReduxNumericInputField = ({ name, min, max, placeholder, step }) => (
    <Field
        key={name}
        name={name}
        placeholder={placeholder}
        parse={parseFloat}
        min={min}
        max={max}
        step={step}
        component={ReduxNumericInput}
        className="border bg-base-100 border-base-300 text-base-600 p-3 pr-8 rounded-r-sm cursor-pointer z-1 focus:border-base-300 w-full font-600"
    />
);

ReduxNumericInputField.propTypes = {
    name: PropTypes.string.isRequired,
    min: PropTypes.number,
    max: PropTypes.number,
    step: PropTypes.number,
    placeholder: PropTypes.string
};

ReduxNumericInputField.defaultProps = {
    min: 1,
    max: Number.MAX_SAFE_INTEGER,
    step: 1,
    placeholder: ''
};

export default ReduxNumericInputField;
