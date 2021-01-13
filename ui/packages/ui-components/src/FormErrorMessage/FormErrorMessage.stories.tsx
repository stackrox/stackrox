import React from 'react';
import { Meta, Story } from '@storybook/react/types-6-0';
import { Formik, Form } from 'formik';

import FormErrorMessage from './FormErrorMessage';

export default {
    title: 'FormErrorMessage',
    component: FormErrorMessage,
} as Meta;

export const Default: Story = () => {
    const initialErrors = { name: 'Required' };
    const initialTouched = { name: true };
    function onSubmit(): void {}
    return (
        <Formik
            initialValues={{}}
            initialErrors={initialErrors}
            initialTouched={initialTouched}
            onSubmit={onSubmit}
        >
            <Form>
                <FormErrorMessage name="name" />
            </Form>
        </Formik>
    );
};

export const NoError: Story = () => {
    const initialErrors = {};
    const initialTouched = {};
    function onSubmit(): void {}
    return (
        <Formik
            initialValues={{}}
            initialErrors={initialErrors}
            initialTouched={initialTouched}
            onSubmit={onSubmit}
        >
            <Form>
                <FormErrorMessage name="name" />
            </Form>
        </Formik>
    );
};
