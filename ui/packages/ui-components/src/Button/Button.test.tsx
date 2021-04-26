import React from 'react';
import { render, screen } from '@testing-library/react';

import Button from './Button';

describe('Button', () => {
    test('renders title, subtitle and footer', () => {
        function onClick(): void {}

        render(
            <Button type="button" onClick={onClick} colorType="alert">
                Click me!
            </Button>
        );

        expect(screen.getByText('Click me!')).toBeInTheDocument();
    });
});
