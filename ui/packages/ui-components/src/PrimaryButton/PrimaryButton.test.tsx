import React from 'react';
import { render, screen } from '@testing-library/react';

import PrimaryButton from './PrimaryButton';

describe('PrimaryButton', () => {
    it('renders', () => {
        function onClick(): void {}

        render(
            <PrimaryButton type="button" onClick={onClick}>
                Save cluster
            </PrimaryButton>
        );

        expect(screen.getByText('Save cluster')).toBeInTheDocument();
    });

    it('has the correct style', () => {
        function onClick(): void {}

        const { container } = render(
            <PrimaryButton type="button" onClick={onClick}>
                Click me!
            </PrimaryButton>
        );

        expect(container).toMatchSnapshot();
    });
});
