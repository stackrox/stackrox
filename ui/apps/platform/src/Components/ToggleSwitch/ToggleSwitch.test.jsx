import React from 'react';
import { fireEvent, render, screen } from '@testing-library/react';

import ToggleSwitch from 'Components/ToggleSwitch';

describe('ToggleSwitch', () => {
    const id = 'enableAutoUpgrade';
    const label = 'Automatically upgrade secured clusters';

    it('should have a label', () => {
        // arrange
        const toggleHandler = vi.fn();
        render(<ToggleSwitch id={id} toggleHandler={toggleHandler} label={label} />);

        // act
        const inputElement = screen.getByLabelText(label);

        // assert
        expect(inputElement).toHaveAttribute('type', 'checkbox');
    });

    it('should add any given extra classes to its root element', () => {
        // arrange
        const toggleHandler = vi.fn();
        const extraClassNames = 'toggle-switch-alert';
        const { container } = render(
            <ToggleSwitch
                id={id}
                toggleHandler={toggleHandler}
                label={label}
                extraClassNames={extraClassNames}
            />
        );

        // act
        // eslint-disable-next-line testing-library/no-node-access
        const rootElement = container.firstChild;

        // assert
        expect(rootElement).toHaveClass('toggle-switch-wrapper');
        expect(rootElement).toHaveClass(extraClassNames);
    });

    it('should set its `checked` prop to false if it does not have an `enabled` prop', () => {
        // arrange
        const toggleHandler = vi.fn();
        render(<ToggleSwitch id={id} toggleHandler={toggleHandler} label={label} />);

        // act
        const inputElement = screen.getByRole('checkbox');

        // assert
        expect(inputElement).not.toBeChecked();
    });

    it('should set its `checked` prop to true it has an `enabled` prop', () => {
        // arrange
        const toggleHandler = vi.fn();
        render(<ToggleSwitch id={id} toggleHandler={toggleHandler} label={label} enabled />);

        // act
        const inputElement = screen.getByRole('checkbox');

        // assert
        expect(inputElement).toBeChecked();
    });

    it('should call its `toggleHandler` prop when the checkbox is clicked', () => {
        // arrange
        const toggleHandler = vi.fn();
        render(<ToggleSwitch id={id} toggleHandler={toggleHandler} label={label} />);

        // act
        fireEvent.click(screen.getByLabelText(label));

        // assert
        expect(toggleHandler).toHaveBeenCalled();
    });
});
