import React, { ReactNode } from 'react';
import { Dropdown, DropdownToggle, Button, FocusTrap } from '@patternfly/react-core';
import { TimesIcon } from '@patternfly/react-icons';

import useSelectToggle from 'hooks/patternfly/useSelectToggle';

export type OptionsDropdownProps = {
    children: ReactNode;
    toggleId: string;
};

/**
 * Component that uses a PatternFly `Dropdown` with the contents wrapped in a `FocusTrap` to
 * enhance accessibility for interactive elements. Upon opening the dropdown, the browser will focus
 * on the container's close button and only allow tab navigation within the container.
 */
function OptionsDropdown({ children, toggleId }: OptionsDropdownProps) {
    const { isOpen, onToggle } = useSelectToggle();

    function handleToggle(isDropdownOpen, event) {
        if (event.key !== 'Tab') {
            onToggle(isDropdownOpen);
        }
    }

    return (
        <Dropdown
            className="pf-u-mr-sm"
            toggle={
                <DropdownToggle id={toggleId} toggleVariant="secondary" onToggle={handleToggle}>
                    Options
                </DropdownToggle>
            }
            position="right"
            isOpen={isOpen}
        >
            <FocusTrap
                focusTrapOptions={{
                    // If there is no focusable element passed in `props.children`, this will put
                    // the focus on the toggle button instead of throwing an error.
                    fallbackFocus: `#${toggleId}`,
                    clickOutsideDeactivates: true,
                    escapeDeactivates: true,
                }}
            >
                <Button
                    style={{ position: 'absolute', top: '0.75rem', right: 0 }}
                    variant="plain"
                    aria-label="Close options"
                    onClick={() => onToggle(false)}
                >
                    <TimesIcon />
                </Button>
                <div className="pf-u-pr-md">{children}</div>
            </FocusTrap>
        </Dropdown>
    );
}

export default OptionsDropdown;
