import React from 'react';
import { fireEvent, getNodeText, render, screen } from '@testing-library/react';

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

    it('should show the placeholder as the first option', () => {
        // arrange
        render(
            <Select
                options={initialOptions}
                placeholder={initialPlaceholder}
                onChange={jest.fn()}
            />
        );

        // act
        const firstOption = screen.getAllByRole('option')[0];

        // assert
        expect(firstOption).toBeInTheDocument();
        expect(getNodeText(firstOption)).toEqual(initialPlaceholder);
    });

    it('should have option elements for every option object, plus the placeholder', () => {
        // arrange
        render(
            <Select
                options={initialOptions}
                placeholder={initialPlaceholder}
                onChange={jest.fn()}
            />
        );

        // act
        const optionElements = screen.getAllByRole('option');

        // assert
        expect(optionElements.length).toEqual(initialOptions.length + 1);
    });

    it('should pass the option clicked on to its provided handler', () => {
        // arrange
        const selectedOptionObject = initialOptions[1];
        const onChangeSpy = jest.fn();
        render(
            <Select
                options={initialOptions}
                placeholder={initialPlaceholder}
                onChange={onChangeSpy}
            />
        );

        // act
        const mockChangeEvent = {
            target: { value: selectedOptionObject.value },
        };
        fireEvent.change(screen.getByRole('combobox'), mockChangeEvent);

        // assert
        expect(onChangeSpy).toHaveBeenCalledWith(selectedOptionObject);
    });
});
