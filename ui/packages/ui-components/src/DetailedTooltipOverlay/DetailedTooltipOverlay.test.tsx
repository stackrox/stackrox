import React from 'react';
import { render, screen } from '@testing-library/react';

import DetailedTooltipOverlay from './DetailedTooltipOverlay';

describe('DetailedTooltipOverlay', () => {
    test('renders title, subtitle and footer', () => {
        render(
            <DetailedTooltipOverlay
                title="Title"
                subtitle="Subtitle"
                body={<p className="my-class">Body</p>}
                footer="Footer"
            />
        );
        expect(screen.getByTestId('tooltip-title')).toHaveTextContent('Title');
        expect(screen.getByTestId('tooltip-subtitle')).toHaveTextContent('Subtitle');
        expect(screen.getByTestId('tooltip-footer')).toHaveTextContent('Footer');

        expect(screen.getByTestId('tooltip-body')).toHaveTextContent('Body');
        expect(screen.getByText('Body')).toHaveClass('my-class');
    });
});
