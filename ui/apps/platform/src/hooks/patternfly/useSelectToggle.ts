import { useCallback, useState } from 'react';

interface SelectToggleReturn {
    /** Whether or not the Select component should be displayed as open */
    isOpen: boolean;
    /** Callback that fires when the toggle element is clicked in the component */
    onToggle: (
        // TODO: add this as a param once all Selects are using PF5
        // event:
        //     | Event
        //     | React.MouseEvent<Element, MouseEvent>
        //     | React.ChangeEvent<Element>
        //     | React.KeyboardEvent<Element>,
        isExpanded: boolean
    ) => void;
    /** Function that sets the toggle state of the component */
    toggleSelect: (
        // TODO: add this as a param once all Selects are using PF5
        // event:
        //     | Event
        //     | React.MouseEvent<Element, MouseEvent>
        //     | React.ChangeEvent<Element>
        //     | React.KeyboardEvent<Element>,
        isExpanded: boolean
    ) => void;
    /** Function that opens the component dropdown */
    openSelect: () => void;
    /** Function that closes the component dropdown */
    closeSelect: () => void;
}

/**
 * Hook to aid in handling Select component dropdown state, especially in
 * PatternFly Select components.
 */
function useSelectToggle(defaultExpanded = false): SelectToggleReturn {
    const [isOpen, setIsOpen] = useState<boolean>(defaultExpanded);
    const onToggle = useCallback(() => setIsOpen(!isOpen), [isOpen, setIsOpen]);
    const toggleSelect = useCallback(setIsOpen, [setIsOpen]);
    const openSelect = useCallback(() => toggleSelect(true), [toggleSelect]);
    const closeSelect = useCallback(() => toggleSelect(false), [toggleSelect]);

    return { isOpen, onToggle, toggleSelect, openSelect, closeSelect };
}

export default useSelectToggle;
