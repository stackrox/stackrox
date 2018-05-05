import React from 'react';
import PropTypes from 'prop-types';

import FormFieldRemoveButton from 'Components/FormFieldRemoveButton';

const FormField = props => (
    <div className="mb-4 transition">
        <div className="py-2 text-primary-500">
            <span>{props.label}</span>
            {props.required ? <span className="required text-danger-500 ml-2">*</span> : ''}
        </div>
        <div className="flex">
            {props.children}
            {props.onRemove && (
                <FormFieldRemoveButton field={props.value} onClick={props.onRemove} />
            )}
        </div>
    </div>
);

FormField.defaultProps = {
    onRemove: null,
    value: ''
};

FormField.propTypes = {
    children: PropTypes.node.isRequired,
    onRemove: PropTypes.func,
    label: PropTypes.string.isRequired,
    value: PropTypes.string,
    required: PropTypes.bool.isRequired
};

export default FormField;
