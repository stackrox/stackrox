import React, { ReactElement, useState } from 'react';
import { Meta, Story } from '@storybook/react/types-6-0';
import * as yup from 'yup';
import { Formik, Form, useFormikContext, FormikValues } from 'formik';

import CondensedButton from '../CondensedButton';
import FormTextInput, { OnChangeHandler } from './FormTextInput';

export default {
    title: 'FormTextInput',
    component: FormTextInput,
} as Meta;

interface SingleFormValues {
    name: string;
}

interface MultiFormValues {
    firstName: string;
    lastName: string;
    nickname: string;
}

interface FormValuesDisplayProps {
    formValues: SingleFormValues | MultiFormValues;
}

interface FormSubmitButtonProps {
    children: ReactElement[] | ReactElement | string;
}

function FormSubmitButton({ children }: FormSubmitButtonProps): ReactElement {
    const { submitForm } = useFormikContext<FormikValues>();
    return (
        <CondensedButton type="submit" onClick={submitForm}>
            {children}
        </CondensedButton>
    );
}

const FormValuesDisplay = ({ formValues }: FormValuesDisplayProps): ReactElement => {
    return <pre className="bg-base-200 mt-4 p-4">{JSON.stringify(formValues)}</pre>;
};

export const Disabled: Story = (): ReactElement => {
    const [formValues, setFormValues] = useState<SingleFormValues>();

    const initialValues: SingleFormValues = { name: '' };
    const onSubmit = (values: SingleFormValues): void => {
        setFormValues(values);
    };

    return (
        <>
            <Formik initialValues={initialValues} onSubmit={onSubmit}>
                <Form>
                    <FormTextInput label="Name" name="name" isDisabled />
                    <div className="mt-4">
                        <FormSubmitButton>Submit</FormSubmitButton>
                    </div>
                </Form>
            </Formik>
            <FormValuesDisplay formValues={formValues} />
        </>
    );
};

export const Validation: Story = (): ReactElement => {
    const [formValues, setFormValues] = useState<SingleFormValues>();

    const initialValues: SingleFormValues = { name: '' };
    const initialErrors = { name: 'Name is required' };
    const initialTouched = { name: true };
    const validationSchema = yup.object().shape({
        name: yup.string().max(15, 'Must be 15 characters or less').required('Name is required'),
    });
    const onSubmit = (values: SingleFormValues): void => {
        setFormValues(values);
    };
    return (
        <>
            <Formik
                initialValues={initialValues}
                initialErrors={initialErrors}
                initialTouched={initialTouched}
                validationSchema={validationSchema}
                onSubmit={onSubmit}
            >
                <Form>
                    <FormTextInput label="Name" name="name" />
                    <div className="mt-4">
                        <FormSubmitButton>Submit</FormSubmitButton>
                    </div>
                </Form>
            </Formik>
            <FormValuesDisplay formValues={formValues} />
        </>
    );
};

export const Required: Story = (): ReactElement => {
    const [formValues, setFormValues] = useState<SingleFormValues>();

    const initialValues: SingleFormValues = { name: '' };
    const validationSchema = yup.object().shape({
        name: yup.string().required('Name is required'),
    });
    const onSubmit = (values: SingleFormValues): void => {
        setFormValues(values);
    };

    return (
        <>
            <Formik
                initialValues={initialValues}
                validationSchema={validationSchema}
                onSubmit={onSubmit}
            >
                <Form>
                    <FormTextInput label="Name" name="name" isRequired />
                    <div className="mt-4">
                        <FormSubmitButton>Submit</FormSubmitButton>
                    </div>
                </Form>
            </Formik>
            <FormValuesDisplay formValues={formValues} />
        </>
    );
};

export const ChangeHandler: Story = (): ReactElement => {
    const [formValues, setFormValues] = useState<SingleFormValues>();

    const initialValues: SingleFormValues = { name: '' };
    const onChange: OnChangeHandler = ({ event, handleChange }) => {
        if (event.target.value === 'Shazam') {
            const modifiedEvent = { ...event };
            modifiedEvent.target.value = '';
            handleChange(modifiedEvent);
        }
        handleChange(event);
    };
    const onSubmit = (values: SingleFormValues): void => {
        setFormValues(values);
    };

    return (
        <>
            <Formik initialValues={initialValues} onSubmit={onSubmit}>
                <Form>
                    <FormTextInput
                        label="Name"
                        name="name"
                        helperText="The changeHandler will clear the input if you write 'Shazam'"
                        onChange={onChange}
                    />
                    <div className="mt-4">
                        <FormSubmitButton>Submit</FormSubmitButton>
                    </div>
                </Form>
            </Formik>
            <FormValuesDisplay formValues={formValues} />
        </>
    );
};

export const Multiple: Story = (): ReactElement => {
    const [formValues, setFormValues] = useState<MultiFormValues>();
    const initialValues: MultiFormValues = { firstName: '', lastName: '', nickname: '' };
    const onSubmit = (values: MultiFormValues): void => {
        setFormValues(values);
    };

    return (
        <>
            <Formik initialValues={initialValues} onSubmit={onSubmit}>
                <Form>
                    <div className="mb-4">
                        <FormTextInput label="First Name" name="firstName" />
                    </div>
                    <div className="mb-4">
                        <FormTextInput label="Last Name" name="lastName" />
                    </div>
                    <FormTextInput label="Nickname" name="nickname" />
                    <div className="mt-4">
                        <FormSubmitButton>Submit</FormSubmitButton>
                    </div>
                </Form>
            </Formik>
            <FormValuesDisplay formValues={formValues} />
        </>
    );
};
