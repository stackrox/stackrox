import React, { ReactElement } from 'react';
import PropTypes, { InferProps } from 'prop-types';

function FormLabel({ label, helperText, isRequired, children }: FormLabelProps): ReactElement {
    return (
        <label className="flex flex-col">
            <div className="flex items-center font-600 text-base-700">
                {label}{' '}
                {isRequired && (
                    <span
                        className="flex items-center text-alert-500 text-sm ml-2"
                        aria-label="required"
                    >
                        (required)
                    </span>
                )}
            </div>
            {helperText && <div className="mt-2 text-base text-base-600">{helperText}</div>}
            {children}
        </label>
    );
}

FormLabel.propTypes = {
    label: PropTypes.string.isRequired,
    helperText: PropTypes.string,
    isRequired: PropTypes.bool,
    children: PropTypes.node.isRequired,
};

FormLabel.defaultProps = {
    helperText: null,
    isRequired: false,
} as FormLabelProps;

export type FormLabelProps = InferProps<typeof FormLabel.propTypes>;
export default FormLabel;
