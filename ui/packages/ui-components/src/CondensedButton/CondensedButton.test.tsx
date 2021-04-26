import React from 'react';
import { render, screen } from '@testing-library/react';

import CondensedButton from './CondensedButton';

describe('CondensedButton', () => {
    test('renders the button', () => {
        function onClick(): void {}

        const { container } = render(
            <CondensedButton type="button" onClick={onClick}>
                Click me!
            </CondensedButton>
        );

        expect(screen.getByText('Click me!')).toBeInTheDocument();
        expect(container).toMatchSnapshot();
    });
});
