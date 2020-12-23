import React from 'react';
import { render } from '@testing-library/react';

import CondensedButton from './CondensedButton';

describe('CondensedButton', () => {
    test('renders the button', () => {
        function onClick(): void {}

        const { container, getByText } = render(
            <CondensedButton type="button" onClick={onClick}>
                Click me!
            </CondensedButton>
        );

        expect(getByText('Click me!')).toBeInTheDocument();
        expect(container).toMatchSnapshot();
    });
});
