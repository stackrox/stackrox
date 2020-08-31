import React from 'react';
import { render } from '@testing-library/react';

import HoverHint from './HoverHint';

describe('HoverHint', () => {
    // TODO-ivan: remove this hack once jest (and jsdom) are upgraded, see https://github.com/mui-org/material-ui/issues/15726
    beforeEach(() => {
        document.createRange = (): Range => ({
            setStart: (): void => {},
            setEnd: (): void => {},
            // eslint-disable-next-line @typescript-eslint/ban-ts-ignore
            // @ts-ignore Not going to implement the real DOM here with 47 properties
            commonAncestorContainer: {
                nodeName: 'BODY',
                ownerDocument: document,
            },
        });
    });

    test('adds hint to the element', () => {
        const divEl = document.createElement('div');
        const { getByText } = render(<HoverHint target={divEl}>Hint</HoverHint>);
        expect(getByText('Hint')).toBeInTheDocument();
    });
});
