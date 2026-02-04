import { useState } from 'react';
import type { MouseEvent as ReactMouseEvent } from 'react';

function useSelectToggleState(onSelectionChange: (value: string) => void) {
    const [isOpen, setIsOpen] = useState(false);

    const onSelect = (
        _event: ReactMouseEvent<Element, MouseEvent> | undefined,
        value: string | number | undefined
    ) => {
        if (typeof value === 'string') {
            setIsOpen(false);
            onSelectionChange(value);
        }
    };

    const onToggle = () => {
        setIsOpen((prev) => !prev);
    };

    return {
        isOpen,
        setIsOpen,
        onSelect,
        onToggle,
    };
}

export default useSelectToggleState;
