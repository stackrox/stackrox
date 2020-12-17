import React from 'react';
import { render } from '@testing-library/react';

import FormLabel from './FormLabel';

describe('FormLabel', () => {
    test('shows the label text', () => {
        const { getByText } = render(
            <FormLabel label="Name">
                <input type="text" className="form-input mt-3 bg-base-200" disabled />
            </FormLabel>
        );
        expect(getByText('Name')).toBeInTheDocument();
    });

    test('shows the helper text', () => {
        const { getByText } = render(
            <FormLabel label="Name" helperText="This is some helper text">
                <input type="text" className="form-input mt-3 bg-base-200" disabled />
            </FormLabel>
        );
        expect(getByText('This is some helper text')).toBeInTheDocument();
    });

    test('shows the required text when "isRequired" is true', () => {
        const { getByText } = render(
            <FormLabel label="Name" isRequired>
                <input type="text" className="form-input mt-3 bg-base-200" disabled />
            </FormLabel>
        );
        expect(getByText(/required/)).toBeInTheDocument();
    });
});
