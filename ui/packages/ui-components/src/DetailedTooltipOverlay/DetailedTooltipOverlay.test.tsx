import React from 'react';
import { render } from '@testing-library/react';

import DetailedTooltipOverlay from './DetailedTooltipOverlay';

describe('DetailedTooltipOverlay', () => {
    test('renders title, subtitle and footer', () => {
        const { getByText, getByTestId } = render(
            <DetailedTooltipOverlay
                title="Title"
                subtitle="Subtitle"
                body={<p className="my-class">Body</p>}
                footer="Footer"
            />
        );
        expect(getByTestId('tooltip-title')).toHaveTextContent('Title');
        expect(getByTestId('tooltip-subtitle')).toHaveTextContent('Subtitle');
        expect(getByTestId('tooltip-footer')).toHaveTextContent('Footer');

        expect(getByTestId('tooltip-body')).toHaveTextContent('Body');
        expect(getByText('Body')).toHaveClass('my-class');
    });
});
