import React from 'react';
import { render, screen } from '@testing-library/react';

import SuccessButton from './SuccessButton';

describe('SuccessButton', () => {
    it('renders', () => {
        function onClick(): void {}

        render(
            <SuccessButton type="button" onClick={onClick}>
                Save cluster
            </SuccessButton>
        );

        expect(screen.getByText('Save cluster')).toBeInTheDocument();
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
