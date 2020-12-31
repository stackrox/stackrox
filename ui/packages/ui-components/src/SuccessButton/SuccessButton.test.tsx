import React from 'react';
import { render } from '@testing-library/react';

import SuccessButton from './SuccessButton';

describe('SuccessButton', () => {
    it('renders', () => {
        function onClick(): void {}

        const { getByText } = render(
            <SuccessButton type="button" onClick={onClick}>
                Save cluster
            </SuccessButton>
        );

        expect(getByText('Save cluster')).toBeInTheDocument();
    });

    it('has the correct style', () => {
        function onClick(): void {}

        const { container } = render(
            <SuccessButton type="button" onClick={onClick}>
                Click me!
            </SuccessButton>
        );

        expect(container).toMatchSnapshot();
    });
});
