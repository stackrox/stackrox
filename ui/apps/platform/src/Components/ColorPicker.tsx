import { useState } from 'react';
import { Popover } from '@patternfly/react-core';
import { ChromePicker } from 'react-color';

type ColorPickerProps = {
    id: string;
    label: string;
    color?: string | null;
    onChange?: (color: string, id: string) => void;
    disabled?: boolean;
};

function ColorPicker({
    id,
    label,
    color = null,
    onChange = () => {},
    disabled = false,
}: ColorPickerProps) {
    const [isOpen, setIsOpen] = useState(false);

    function handleOnChange({ hex }: { hex: string }) {
        onChange(hex, id);
    }

    return (
        <Popover
            aria-label={label}
            hasNoPadding
            hasAutoWidth
            showClose={false}
            isVisible={isOpen}
            shouldOpen={() => !disabled && setIsOpen(true)}
            shouldClose={() => setIsOpen(false)}
            bodyContent={<ChromePicker color={color ?? undefined} onChange={handleOnChange} />}
        >
            <button
                type="button"
                id={id}
                aria-label={label}
                className={`p-1 h-5 w-full border border-base-300 ${
                    disabled ? 'pointer-events-none' : ''
                }`}
            >
                <div style={{ backgroundColor: color ?? undefined }} className="h-full w-full" />
            </button>
        </Popover>
    );
}

export default ColorPicker;
