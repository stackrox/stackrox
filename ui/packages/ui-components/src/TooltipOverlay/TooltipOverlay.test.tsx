import React from 'react';
import { render } from '@testing-library/react';

import TooltipOverlay from './TooltipOverlay';

describe('TooltipOverlay', () => {
    test('renders with proper class', () => {
        const { getByText } = render(<TooltipOverlay>Tooltip</TooltipOverlay>);
        expect(getByText('Tooltip')).toHaveClass('rox-tooltip-overlay');
    });

    test('supports adding extra class', () => {
        const { getByText } = render(
            <TooltipOverlay extraClassName="my-class">Tooltip</TooltipOverlay>
        );
        expect(getByText('Tooltip')).toHaveClass('rox-tooltip-overlay');
        expect(getByText('Tooltip')).toHaveClass('my-class');
    });
});
