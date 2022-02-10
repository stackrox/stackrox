/* eslint-disable no-void */
import { useState } from 'react';

type UseMultiSelect = {
    isOpen: boolean;
    onToggle: () => void;
    onSelect: (e, selection) => void;
    onClear: (event) => void;
};

function useMultiSelect(
    onChange: (selection, event) => void,
    values: string[],
    closeOnSelect = true
): UseMultiSelect {
    const [isOpen, setIsOpen] = useState(false);

    function onToggle() {
        setIsOpen(!isOpen);
    }

    function onSelect(_event, selection) {
        if (values.includes(selection)) {
            const newSelection = values.filter((item) => item !== selection);
            void onChange(newSelection, _event);
        } else {
            const newSelection = [...values, selection];
            onChange(newSelection, _event);
        }

        if (closeOnSelect) {
            setIsOpen(false);
        }
    }

    function onClear(_event) {
        onChange([], _event);
    }

    return {
        isOpen,
        onToggle,
        onSelect,
        onClear,
    };
}

export default useMultiSelect;
