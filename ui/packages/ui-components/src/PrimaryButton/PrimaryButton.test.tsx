import React from 'react';
import { render } from '@testing-library/react';

import PrimaryButton from './PrimaryButton';

describe('PrimaryButton', () => {
    it('renders', () => {
        function onClick(): void {}

        const { getByText } = render(
            <PrimaryButton type="button" onClick={onClick}>
                Save cluster
            </PrimaryButton>
        );

        expect(getByText('Save cluster')).toBeInTheDocument();
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
