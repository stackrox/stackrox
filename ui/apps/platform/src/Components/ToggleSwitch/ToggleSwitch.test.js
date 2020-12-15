import React from 'react';
import { fireEvent, render } from '@testing-library/react';

import ToggleSwitch from 'Components/ToggleSwitch';

describe('ToggleSwitch', () => {
    const id = 'enableAutoUpgrade';
    const label = 'Automatically upgrade secured clusters';

    it('should have a label', () => {
        // arrange
        const toggleHandler = jest.fn();
        const { getByLabelText } = render(
            <ToggleSwitch id={id} toggleHandler={toggleHandler} label={label} />
        );

        // act
        const inputElement = getByLabelText(label);

        // assert
        expect(inputElement).toHaveAttribute('type', 'checkbox');
    });

    it('should add any given extra classes to its root element', () => {
        // arrange
        const toggleHandler = jest.fn();
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
        const rootElement = container.firstChild;

        // assert
        expect(rootElement).toHaveClass('toggle-switch-wrapper');
        expect(rootElement).toHaveClass(extraClassNames);
    });

    it('should set its `checked` prop to false if it does not have an `enabled` prop', () => {
        // arrange
        const toggleHandler = jest.fn();
        const { getByRole } = render(
            <ToggleSwitch id={id} toggleHandler={toggleHandler} label={label} />
        );

        // act
        const inputElement = getByRole('checkbox');

        // assert
        expect(inputElement).not.toBeChecked();
    });

    it('should set its `checked` prop to true it has an `enabled` prop', () => {
        // arrange
        const toggleHandler = jest.fn();
        const { getByRole } = render(
            <ToggleSwitch id={id} toggleHandler={toggleHandler} label={label} enabled />
        );

        // act
        const inputElement = getByRole('checkbox');

        // assert
        expect(inputElement).toBeChecked();
    });

    it('should call its `toggleHandler` prop when the checkbox is clicked', () => {
        // arrange
        const toggleHandler = jest.fn();
        const { getByLabelText } = render(
            <ToggleSwitch id={id} toggleHandler={toggleHandler} label={label} />
        );

        // act
        fireEvent.click(getByLabelText(label));

        // assert
        expect(toggleHandler).toHaveBeenCalled();
    });
});
