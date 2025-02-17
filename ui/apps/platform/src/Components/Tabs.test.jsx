import React from 'react';
import { fireEvent, render, screen } from '@testing-library/react';

import Tabs from 'Components/Tabs';

describe('Component:Tabs', () => {
    it('shows the first tab as the active tab', () => {
        const headers = [
            { text: 'Tab 1', disabled: false },
            { text: 'Tab 2', disabled: false },
        ];
        render(<Tabs headers={headers} />);
        const buttons = screen.getAllByRole('button');

        expect(buttons.length).toEqual(headers.length);
        expect(buttons[0]).toHaveClass('active');
        expect(buttons[1]).not.toHaveClass('active');
    });

    it('should be able to switch active tabs', () => {
        const headers = [
            { text: 'Tab 1', disabled: false },
            { text: 'Tab 2', disabled: false },
            { text: 'Tab 3', disabled: false },
        ];
        render(<Tabs headers={headers} />);
        const buttons = screen.getAllByRole('button');

        fireEvent.click(buttons[1]);
        expect(buttons[0]).not.toHaveClass('active');
        expect(buttons[1]).toHaveClass('active');

        fireEvent.click(buttons[2]);
        expect(buttons[1]).not.toHaveClass('active');
        expect(buttons[2]).toHaveClass('active');
    });

    it('should not be able to switch to a disabled tab', () => {
        const headers = [
            { text: 'Tab 1', disabled: false },
            { text: 'Tab 2', disabled: true },
            { text: 'Tab 3', disabled: false },
        ];
        render(<Tabs headers={headers} />);
        const buttons = screen.getAllByRole('button');

        fireEvent.click(buttons[1]);
        expect(buttons[0]).toHaveClass('active');
        expect(buttons[1]).not.toHaveClass('active');

        fireEvent.click(buttons[2]);
        expect(buttons[0]).not.toHaveClass('active');
        expect(buttons[2]).toHaveClass('active');
    });
});
