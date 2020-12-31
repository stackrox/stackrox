import React, { ReactElement } from 'react';

import Button, { HOCButtonProps } from '../Button';

function SuccessButton({ type = 'button', onClick, children }: HOCButtonProps): ReactElement {
    return (
        <Button colorType="success" type={type} onClick={onClick}>
            {children}
        </Button>
    );
}

export default SuccessButton;
