import React, { ReactElement } from 'react';
import { render, screen } from '@testing-library/react';
import { Formik, Form } from 'formik';

import FormErrorMessage from './FormErrorMessage';

const FormErrorMessageTestComponent = (): ReactElement => {
    const initialErrors = { name: 'Looks like you messed up' };
    const initialTouched = { name: 'Looks like you messed up' };
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

describe('FormErrorMessage', () => {
    test('renders title, subtitle and footer', () => {
        render(<FormErrorMessageTestComponent />);

        expect(screen.getByText('Looks like you messed up')).toBeInTheDocument();
    });
});
