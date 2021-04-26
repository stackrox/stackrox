import React from 'react';
import { render, screen } from '@testing-library/react';

import CondensedAlertButton from './CondensedAlertButton';

describe('CondensedAlertButton', () => {
    test('renders the button', () => {
        function onClick(): void {}

        const { container } = render(
            <CondensedAlertButton type="button" onClick={onClick}>
                Click me!
            </CondensedAlertButton>
        );

        expect(screen.getByText('Click me!')).toBeInTheDocument();
        expect(container).toMatchSnapshot();
    });
});
