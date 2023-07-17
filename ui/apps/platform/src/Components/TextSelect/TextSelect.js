import React from 'react';
import { ChevronDown } from 'react-feather';

import Select from 'Components/ReactSelect';

const DropdownIndicator = () => {
    return (
        <ChevronDown className="h-5 w-5 text-primary-800 border-2 border-base-400 rounded-full" />
    );
};

const TextSelect = ({ ...rest }) => {
    const { options } = { ...rest };
    if (options.length === 1) {
        return options[0].label;
    }
    const selectStyles = {
        valueContainer: (base) => ({
            ...base,
            'padding-left': '0',
        }),
        control: (base) => ({
            ...base,
            border: 'none',
            'font-weight': '700!important',
            color: 'var(--base-600)',
            cursor: 'pointer !important',
        }),
        indicatorSeparator: (base) => ({ ...base, display: 'none' }),
    };
    const components = {
        DropdownIndicator,
    };
    // Because React Select renders a dummy input element,
    // axe DevTools reports a theoretical issue: Form elements must have labels.
    // One of its suggestions is to enclose the form element in a label element.
    // Thankfully that does not affect the layout.
    // However, jsx-a11y rule only accepts label as sibling, not enclosing.
    /* eslint-disable jsx-a11y/label-has-associated-control */
    return (
        <label>
            <Select styles={selectStyles} isSearchable={false} {...rest} components={components} />
        </label>
    );
    /* eslint-enable jsx-a11y/label-has-associated-control */
};

export default TextSelect;
