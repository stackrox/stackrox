import React from 'react';
import { render, screen } from '@testing-library/react';

import HoverHint from './HoverHint';

describe('HoverHint', () => {
    test('adds hint to the element', () => {
        const divEl = document.createElement('div');
        render(<HoverHint target={divEl}>Hint</HoverHint>);
        expect(screen.getByText('Hint')).toBeInTheDocument();
    });
});
