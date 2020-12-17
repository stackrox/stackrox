import React from 'react';
import { render } from '@testing-library/react';

import Button from './Button';

describe('Button', () => {
    test('renders title, subtitle and footer', () => {
        function onClick(): void {}

        const { getByText } = render(
            <Button type="button" onClick={onClick}>
                Click me!
            </Button>
        );

        expect(getByText('Click me!')).toBeInTheDocument();
    });
});
