import { useState } from 'react';

/**
 * Custom hook for managing common select component state and handlers.
 * Provides standard state management for PatternFly v5 Select components.
 */
function useSelectState(onSelectionChange: (value: string) => void) {
    const [isOpen, setIsOpen] = useState(false);

    const onSelect = (
        _event: React.MouseEvent<Element, MouseEvent> | undefined,
        value: string | number | undefined
    ) => {
        if (typeof value === 'string') {
            setIsOpen(false);
            onSelectionChange(value);
        }
    };

    const onToggle = () => {
        setIsOpen(!isOpen);
    };

    return {
        isOpen,
        setIsOpen,
        onSelect,
        onToggle,
    };
}

export default useSelectState;
