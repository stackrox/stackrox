import React from 'react';
import PropTypes from 'prop-types';

import FormFieldRemoveButton from 'Components/FormFieldRemoveButton';

const FormField = ({ label, required, testId, children, onRemove, name }) => (
    <div className="mb-4">
        <div className="py-2 text-base-600 font-700">
            <span>{label}</span>
            {required && (
                <span data-testid="required" className="text-alert-500 ml-1">
                    *
                </span>
            )}
        </div>
        <div className="flex" data-testid={testId}>
            {children}
            {onRemove && <FormFieldRemoveButton field={name} onClick={onRemove} />}
        </div>
    </div>
);

FormField.defaultProps = {
    name: '',
    required: false,
    onRemove: null,
    testId: '',
};

FormField.propTypes = {
    name: PropTypes.string,
    label: PropTypes.string.isRequired,
    required: PropTypes.bool,
    onRemove: PropTypes.func,
    children: PropTypes.node.isRequired,
    testId: PropTypes.string,
};

export default FormField;
