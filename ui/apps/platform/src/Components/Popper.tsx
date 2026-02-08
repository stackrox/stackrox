import { useState } from 'react';
import type { ReactElement, ReactNode } from 'react';
import { Popover } from '@patternfly/react-core';

type PopperPlacement = 'top' | 'bottom' | 'left' | 'right';

type PopperProps = {
    disabled?: boolean;
    placement?: PopperPlacement;
    buttonClass?: string;
    buttonContent: ReactNode;
    popperContent: ReactElement;
};

/**
 * A popover component that wraps PatternFly's Popover.
 * Displays toggleable content when clicking the trigger button.
 */
function Popper({
    disabled = false,
    placement = 'right',
    buttonClass = '',
    buttonContent,
    popperContent,
}: PopperProps) {
    const [isOpen, setIsOpen] = useState(false);

    return (
        <Popover
            aria-label="Popover"
            hasNoPadding
            hasAutoWidth
            showClose={false}
            position={placement}
            isVisible={isOpen}
            shouldOpen={() => !disabled && setIsOpen(true)}
            shouldClose={() => setIsOpen(false)}
            bodyContent={popperContent}
        >
            <button
                type="button"
                data-testid="popper-button"
                className={`${buttonClass} ${disabled ? 'pointer-events-none' : ''}`}
            >
                {buttonContent}
            </button>
        </Popover>
    );
}

export default Popper;
