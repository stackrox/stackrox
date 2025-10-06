import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { vi } from 'vitest';

import TypeaheadSelect, { TypeaheadSelectOption } from './TypeaheadSelect';

const options: TypeaheadSelectOption[] = [
    { value: 'apple', label: 'Apple' },
    { value: 'banana', label: 'Banana' },
    { value: 'cherry', label: 'Cherry' },
];

describe('TypeaheadSelect', () => {
    it('should select option and call onChange', async () => {
        const onChange = vi.fn();
        render(
            <TypeaheadSelect id="test-select-1" value="" onChange={onChange} options={options} />
        );

        const input = screen.getByPlaceholderText('Type to search...');
        await userEvent.click(input);

        const option = screen.getByText('Apple');
        await userEvent.click(option);

        expect(onChange).toHaveBeenCalledWith('apple');
    });

    it('should navigate with keyboard and select with Enter', async () => {
        const onChange = vi.fn();
        render(
            <TypeaheadSelect id="test-select-2" value="" onChange={onChange} options={options} />
        );

        const input = screen.getByPlaceholderText('Type to search...');
        await userEvent.click(input); // Open dropdown
        await userEvent.keyboard('{ArrowDown}'); // Focus first option
        await userEvent.keyboard('{Enter}'); // Select focused option

        expect(onChange).toHaveBeenCalledWith('apple');
    });

    it('should close dropdown and clear input with Escape', async () => {
        const onChange = vi.fn();
        render(
            <TypeaheadSelect id="test-select-3" value="" onChange={onChange} options={options} />
        );

        const input = screen.getByPlaceholderText('Type to search...');
        await userEvent.type(input, 'App');

        // Verify dropdown is open (filtered option should be visible)
        expect(screen.getByText('Apple')).toBeInTheDocument();

        await userEvent.keyboard('{Escape}');

        expect(input).toHaveValue('');
        // Verify dropdown is closed (options should not be visible)
        expect(screen.queryByText('Apple')).not.toBeInTheDocument();
    });

    it('should filter options based on input', async () => {
        const onChange = vi.fn();
        render(
            <TypeaheadSelect id="test-select-4" value="" onChange={onChange} options={options} />
        );

        const input = screen.getByPlaceholderText('Type to search...');
        await userEvent.type(input, 'App');

        expect(screen.getByText('Apple')).toBeInTheDocument();
        expect(screen.queryByText('Banana')).not.toBeInTheDocument();
        expect(screen.queryByText('Cherry')).not.toBeInTheDocument();
    });

    it('should show create option when allowCreate is true', async () => {
        const onChange = vi.fn();
        render(
            <TypeaheadSelect
                id="test-select-5"
                value=""
                onChange={onChange}
                options={options}
                allowCreate
            />
        );

        const input = screen.getByPlaceholderText('Type to search...');
        await userEvent.type(input, 'orange');

        expect(screen.getByText('Create "orange"')).toBeInTheDocument();
    });

    it('should not show create option when input matches existing option', async () => {
        const onChange = vi.fn();
        render(
            <TypeaheadSelect
                id="test-select-6"
                value=""
                onChange={onChange}
                options={options}
                allowCreate
            />
        );

        const input = screen.getByPlaceholderText('Type to search...');
        await userEvent.type(input, 'Apple');

        // The existing option should be visible
        expect(screen.getByText('Apple')).toBeInTheDocument();
        // But the create option should not be visible since it matches an existing option
        expect(screen.queryByText('Create "Apple"')).not.toBeInTheDocument();
    });
});
