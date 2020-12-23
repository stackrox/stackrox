import React from 'react';
import { render } from '@testing-library/react';

import CondensedAlertButton from './CondensedAlertButton';

describe('CondensedAlertButton', () => {
    test('renders the button', () => {
        function onClick(): void {}

        const { container, getByText } = render(
            <CondensedAlertButton type="button" onClick={onClick}>
                Click me!
            </CondensedAlertButton>
        );

        expect(getByText('Click me!')).toBeInTheDocument();
        expect(container).toMatchSnapshot();
    });
});
