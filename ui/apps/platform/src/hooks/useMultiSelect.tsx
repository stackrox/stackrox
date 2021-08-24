import { useState } from 'react';

type UseMultiSelect = {
    isOpen: boolean;
    onToggle: () => void;
    onSelect: (e, selection) => void;
    onClear: () => void;
};

function useMultiSelect(onChange: (selection) => void, values: string[]): UseMultiSelect {
    const [isOpen, setIsOpen] = useState(false);

    function onToggle() {
        setIsOpen(!isOpen);
    }

    function onSelect(_event, selection) {
        if (values.includes(selection)) {
            const newSelection = values.filter((item) => item !== selection);
            onChange(newSelection);
            setIsOpen(false);
        } else {
            const newSelection = [...values, selection];
            onChange(newSelection);
            setIsOpen(false);
        }
    }

    function onClear() {
        onChange([]);
    }

    return {
        isOpen,
        onToggle,
        onSelect,
        onClear,
    };
}

export default useMultiSelect;
