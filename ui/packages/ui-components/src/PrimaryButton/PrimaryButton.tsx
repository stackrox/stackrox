import React, { ReactElement } from 'react';

import Button, { HOCButtonProps } from '../Button';

function PrimaryButton({
    type = 'button',
    onClick,
    children,
    isDisabled = false,
}: HOCButtonProps): ReactElement {
    return (
        <Button colorType="primary" type={type} onClick={onClick} isDisabled={isDisabled}>
            {children}
        </Button>
    );
}

export default PrimaryButton;
