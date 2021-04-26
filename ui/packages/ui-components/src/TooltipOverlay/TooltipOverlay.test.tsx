import React from 'react';
import { render, screen } from '@testing-library/react';

import TooltipOverlay from './TooltipOverlay';

describe('TooltipOverlay', () => {
    test('renders with proper class', () => {
        render(<TooltipOverlay>Tooltip</TooltipOverlay>);
        expect(screen.getByText('Tooltip')).toHaveClass('rox-tooltip-overlay');
    });

    test('supports adding extra class', () => {
        render(<TooltipOverlay extraClassName="my-class">Tooltip</TooltipOverlay>);
        expect(screen.getByText('Tooltip')).toHaveClass('rox-tooltip-overlay');
        expect(screen.getByText('Tooltip')).toHaveClass('my-class');
    });
});
