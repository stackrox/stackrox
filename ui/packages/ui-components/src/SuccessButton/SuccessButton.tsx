import React, { ReactElement } from 'react';

import Button, { HOCButtonProps } from '../Button';

function SuccessButton({
    type = 'button',
    onClick,
    children,
    isDisabled = false,
}: HOCButtonProps): ReactElement {
    return (
        <Button colorType="success" type={type} onClick={onClick} isDisabled={isDisabled}>
            {children}
        </Button>
    );
}

export default SuccessButton;
