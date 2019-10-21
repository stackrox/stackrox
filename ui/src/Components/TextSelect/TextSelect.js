import React from 'react';
import { ChevronDown } from 'react-feather';

import Select from 'Components/ReactSelect';

const DropdownIndicator = () => {
    return (
        <ChevronDown className="h-5 w-5 text-primary-800 border-2 border-primary-200 rounded-full" />
    );
};

const TextSelect = ({ ...rest }) => {
    const { options } = { ...rest };
    if (options.length === 1) return options[0].label;
    const selectStyles = {
        valueContainer: base => ({
            ...base,
            'padding-left': '0'
        }),
        control: base => ({
            ...base,
            border: 'none',
            'letter-spacing': '.03125rem',
            'text-transform': 'uppercase',
            'font-weight': '700!important',
            color: 'var(--base-600)',
            'font-size': '.6875rem',
            cursor: 'pointer !important'
        }),
        indicatorSeparator: base => ({ ...base, display: 'none' })
    };
    const components = {
        DropdownIndicator
    };
    return <Select styles={selectStyles} isSearchable={false} {...rest} components={components} />;
};

export default TextSelect;
