import React from 'react';
import { mount } from 'enzyme';

import ToggleSwitch from 'Components/ToggleSwitch';

describe('Component:Select', () => {
    const initialProps = {
        id: 'enableAutoUpgrade',
        label: 'Automatically upgrade secured clusters',
        toggleHandler: () => {},
    };

    it('should show its label in a leading <label> element', () => {
        // arrange
        const toggleSwitch = mount(<ToggleSwitch {...initialProps} />);

        // act
        const labelElement = toggleSwitch.find('label').first(); // second label for custom UI

        // assert
        expect(labelElement.text()).toEqual(initialProps.label);
    });

    it('should add any given extra classes to its root element', () => {
        // arrange
        const extraClassNames = 'toggle-switch-alert';
        const modifiedProps = { ...initialProps, extraClassNames };
        const toggleSwitch = mount(<ToggleSwitch {...modifiedProps} />);

        // act
        const wrapperElement = toggleSwitch.find('.toggle-switch-wrapper');

        // assert
        expect(wrapperElement.hasClass(extraClassNames)).toBe(true);
    });

    it('should should set its <checkbox> to false if not passed an `enabled` prop', () => {
        // arrange
        const toggleSwitch = mount(<ToggleSwitch {...initialProps} />);

        // act
        const checkboxElement = toggleSwitch.find('input');

        // assert
        expect(checkboxElement.prop('checked')).toEqual(false);
    });

    it('should should set its <checkbox> to the given `enabled` prop', () => {
        // arrange
        const modifiedProps = { ...initialProps, enabled: true };
        const toggleSwitch = mount(<ToggleSwitch {...modifiedProps} />);

        // act
        const checkboxElement = toggleSwitch.find('input');

        // assert
        expect(checkboxElement.prop('checked')).toEqual(modifiedProps.enabled);
    });

    it('should call its `toggleHandler` prop on change', () => {
        // arrange
        const onChangeSpy = jest.fn();
        const modifiedProps = { ...initialProps, toggleHandler: onChangeSpy };
        const toggleSwitch = mount(<ToggleSwitch {...modifiedProps} />);

        // act
        const checkboxElement = toggleSwitch.find('input');
        checkboxElement.simulate('change', true);

        // assert
        expect(onChangeSpy).toHaveBeenCalled();
    });
});
