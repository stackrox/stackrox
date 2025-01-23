import React, { ReactNode } from 'react';
import { Button } from '@patternfly/react-core';

import { useCIDRFormModal } from './CIDRFormModalProvider';

interface CIDRFormModalButtonProps extends React.ComponentProps<typeof Button> {
    children: ReactNode;
    isInline?: boolean;
    isDisabled?: boolean;
    onOpenCallback?: () => void;
}

function CIDRFormModalButton({
    children,
    isInline = false,
    isDisabled = false,
    onOpenCallback,
    ...props
}: CIDRFormModalButtonProps) {
    const { toggleCIDRFormModal } = useCIDRFormModal();

    function toggleCIDRBlockForm() {
        onOpenCallback?.();
        toggleCIDRFormModal();
    }

    return (
        <Button
            {...props}
            isInline={isInline}
            onClick={toggleCIDRBlockForm}
            isDisabled={isDisabled}
        >
            {children}
        </Button>
    );
}

export default CIDRFormModalButton;
