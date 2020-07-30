import React from 'react';
import { mount } from 'enzyme';

import Select from 'Components/Select';

describe('Component:Select', () => {
    const initialOptions = [
        {
            label: 'AT&T',
            value: 'att',
        },
        {
            label: 'Sprint',
            value: 'sprint',
        },
        {
            label: 'Verizon',
            value: 'verz',
        },
    ];
    const initialPlaceholder = 'Select a service';

    it('shows show the placeholder as the first option', () => {
        // arrange
        const select = mount(
            <Select
                options={initialOptions}
                placeholder={initialPlaceholder}
                onChange={jest.fn()}
            />
        );

        // act
        const firstOption = select.find('option').first();

        // assert
        expect(firstOption.text()).toEqual(initialPlaceholder);
    });

    it('should have option elements for every option object, plus the placeholder', () => {
        // arrange
        const select = mount(
            <Select
                options={initialOptions}
                placeholder={initialPlaceholder}
                onChange={jest.fn()}
            />
        );

        // act
        const optionElements = select.find('option');

        // assert
        expect(optionElements.length).toEqual(initialOptions.length + 1);
    });

    it('should pass the option clicked on to its provided handler', () => {
        // arrange
        const onChangeSpy = jest.fn();
        const select = mount(
            <Select
                options={initialOptions}
                placeholder={initialPlaceholder}
                onChange={onChangeSpy}
            />
        );

        // act
        const mockChangeEvent = {
            target: { value: initialOptions[1].value },
        };
        select.instance().onClick(mockChangeEvent);

        // assert
        expect(onChangeSpy).toHaveBeenCalledWith(initialOptions[1]);
    });
});
