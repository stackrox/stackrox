import React, { ReactElement } from 'react';
import { useFormikContext, FormikValues } from 'formik';
import get from 'lodash.get';

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

type TextInputProps = {
    name: string;
    value?: string;
    placeholder?: string;
    isDisabled?: boolean;
    onChange: React.ChangeEventHandler<HTMLInputElement>;
    onBlur: React.FocusEventHandler<HTMLInputElement>;
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
    const value = get(values, name);
    return (
        <div>
            <FormLabel label={label} helperText={helperText} isRequired={isRequired}>
                <TextInput
                    name={name}
                    value={value}
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

export type FormTextInputProps = {
    label: string;
    name: string;
    helperText?: string;
    placeholder?: string;
    isDisabled?: boolean;
    isRequired?: boolean;
    onChange?: OnChangeHandler;
};

export type OnChangeHandler = (callbackData: {
    name: string;
    event: React.ChangeEvent<HTMLInputElement>;
    handleChange: (event: React.ChangeEvent<HTMLInputElement>) => void;
}) => void;

export default FormTextInput;
