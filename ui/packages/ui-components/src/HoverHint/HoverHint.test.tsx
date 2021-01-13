import React from 'react';
import { render } from '@testing-library/react';

import HoverHint from './HoverHint';

describe('HoverHint', () => {
    test('adds hint to the element', () => {
        const divEl = document.createElement('div');
        const { getByText } = render(<HoverHint target={divEl}>Hint</HoverHint>);
        expect(getByText('Hint')).toBeInTheDocument();
    });
});
