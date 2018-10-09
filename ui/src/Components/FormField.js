import React from 'react';
import PropTypes from 'prop-types';

import FormFieldRemoveButton from 'Components/FormFieldRemoveButton';

const FormField = props => (
    <div className="mb-4">
        <div className="py-2 text-base-600 font-700">
            <span>{props.label}</span>
            {props.required && (
                <span data-test-id="required" className="text-alert-500 ml-1">
                    *
                </span>
            )}
        </div>
        <div className="flex">
            {props.children}
            {props.onRemove && (
                <FormFieldRemoveButton field={props.name} onClick={props.onRemove} />
            )}
        </div>
    </div>
);

FormField.defaultProps = {
    name: '',
    required: false,
    onRemove: null
};

FormField.propTypes = {
    name: PropTypes.string,
    label: PropTypes.string.isRequired,
    required: PropTypes.bool,
    onRemove: PropTypes.func,
    children: PropTypes.node.isRequired
};

export default FormField;
