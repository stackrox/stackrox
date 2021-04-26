import React from 'react';
import { render, screen } from '@testing-library/react';

import FormLabel from './FormLabel';

describe('FormLabel', () => {
    test('shows the label text', () => {
        render(
            <FormLabel label="Name">
                <input type="text" className="form-input mt-3 bg-base-200" disabled />
            </FormLabel>
        );
        expect(screen.getByText('Name')).toBeInTheDocument();
    });

    test('shows the helper text', () => {
        render(
            <FormLabel label="Name" helperText="This is some helper text">
                <input type="text" className="form-input mt-3 bg-base-200" disabled />
            </FormLabel>
        );
        expect(screen.getByText('This is some helper text')).toBeInTheDocument();
    });

    test('shows the required text when "isRequired" is true', () => {
        render(
            <FormLabel label="Name" isRequired>
                <input type="text" className="form-input mt-3 bg-base-200" disabled />
            </FormLabel>
        );
        expect(screen.getByText(/required/)).toBeInTheDocument();
    });
});
