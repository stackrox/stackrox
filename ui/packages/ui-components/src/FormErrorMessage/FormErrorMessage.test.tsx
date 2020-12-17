import React, { ReactElement } from 'react';
import { render } from '@testing-library/react';
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
        const { getByText } = render(<FormErrorMessageTestComponent />);

        expect(getByText('Looks like you messed up')).toBeInTheDocument();
    });
});
