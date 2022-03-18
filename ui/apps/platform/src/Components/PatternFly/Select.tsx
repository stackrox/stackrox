import React, { ChangeEvent, useCallback, useState } from 'react';
import { Select as PFSelect, SelectProps as PFSelectProps } from '@patternfly/react-core';

/**
 * Filters an array of React Elements that have a `value` prop containing
 * a substring of the provided string filter.
 *
 * @param filterValue The string to filter on
 * @param elements An array of ReactElements, or undefined
 */
function filterElementsWithValueProp(
    filterValue: string,
    elements: React.ReactElement[] | undefined
): React.ReactElement[] | undefined {
    if (filterValue === '' || elements === undefined) {
        return elements;
    }

    return elements.filter((reactElement) =>
        reactElement?.props?.value?.toLowerCase().includes(filterValue.toLowerCase())
    );
}

/**
 * This Select component is a thin wrapper around PatternFly's Select component with some default
 * behavior applied. This version will automatically open/close the Select menu when the toggle
 * switch is clicked, as well as automatically close the menu when an Option has been selected.
 */
function Select({
    children,
    onSelect,
    variant,
    ...props
}: Omit<PFSelectProps, 'isOpen' | 'onToggle' | 'onFilter'>) {
    const [isOpen, setIsOpen] = useState<boolean>(false);
    const onToggle = useCallback(() => setIsOpen(!isOpen), [isOpen, setIsOpen]);
    const onFilter = useCallback(
        (e: ChangeEvent<HTMLInputElement> | null, filterValue: string) =>
            filterElementsWithValueProp(filterValue, children),
        [children]
    );

    function onOptionSelect(e, value) {
        if (!variant || variant === 'single') {
            // Auto-close simple Select dropdowns when an option is selected
            setIsOpen(false);
        }
        if (onSelect) {
            onSelect(e, value);
        }
    }

    return (
        <PFSelect
            {...props}
            isOpen={isOpen}
            onToggle={onToggle}
            onFilter={onFilter}
            onSelect={onOptionSelect}
        >
            {children}
        </PFSelect>
    );
}

export default Select;
