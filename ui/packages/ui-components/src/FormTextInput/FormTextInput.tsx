import React, { ReactElement } from 'react';
import PropTypes, { InferProps } from 'prop-types';
import { useFormikContext, FormikValues } from 'formik';

import FormLabel from '../FormLabel';
import FormErrorMessage from '../FormErrorMessage';

function TextInput({
    name,
    value,
    placeholder,
    isDisabled,
    onChange,
    onBlur,
}: TextInputProps): ReactElement {
    return (
        <input
            type="text"
            className={`form-input mt-3 ${isDisabled ? 'bg-base-200' : ''}`}
            id={name}
            name={name}
            value={value || ''}
            disabled={isDisabled || false}
            placeholder={placeholder || undefined}
            onChange={onChange}
            onBlur={onBlur}
        />
    );
}

TextInput.propTypes = {
    name: PropTypes.string.isRequired,
    value: PropTypes.string,
    placeholder: PropTypes.string,
    isDisabled: PropTypes.bool,
    onChange: PropTypes.func.isRequired,
    onBlur: PropTypes.func.isRequired,
};

TextInput.defaultProps = {
    value: '',
    placeholder: '',
    isDisabled: false,
} as TextInputProps;

type TextInputProps = Omit<InferProps<typeof TextInput.propTypes>, 'onChange'> & {
    onChange: React.ChangeEventHandler<HTMLInputElement>;
};

function FormTextInput({
    label,
    name,
    helperText,
    placeholder,
    isDisabled,
    isRequired,
    onChange,
}: FormTextInputProps): ReactElement {
    const { values, handleChange, handleBlur } = useFormikContext<FormikValues>();
    function onChangeHandler(event: React.ChangeEvent<HTMLInputElement>): void {
        if (onChange) {
            onChange({ name, event, handleChange });
        } else {
            handleChange(event);
        }
    }
    return (
        <div>
            <FormLabel label={label} helperText={helperText} isRequired={isRequired}>
                <TextInput
                    name={name}
                    value={values[name]}
                    placeholder={placeholder}
                    isDisabled={isDisabled}
                    onChange={onChangeHandler}
                    onBlur={handleBlur}
                />
            </FormLabel>
            <FormErrorMessage name={name} />
        </div>
    );
}

FormTextInput.propTypes = {
    label: PropTypes.string.isRequired,
    name: PropTypes.string.isRequired,
    helperText: PropTypes.string,
    placeholder: PropTypes.string,
    isDisabled: PropTypes.bool,
    isRequired: PropTypes.bool,
    onChange: PropTypes.func,
};

FormTextInput.defaultProps = {
    helperText: null,
    placeholder: null,
    isDisabled: false,
    isRequired: false,
    onChange: null,
} as FormTextInputProps;

export type OnChangeHandler = (callbackData: {
    name: string;
    event: React.ChangeEvent<HTMLInputElement>;
    handleChange: (event: React.ChangeEvent<HTMLInputElement>) => void;
}) => void;

export type FormTextInputProps = Omit<InferProps<typeof FormTextInput.propTypes>, 'onChange'> & {
    onChange?: OnChangeHandler | null;
};

export default FormTextInput;
