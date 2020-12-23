import React, { ReactElement } from 'react';

import Button, { HOCButtonProps } from '../Button';

export type CondensedButtonProps = HOCButtonProps;

function CondensedButton({ type, onClick, children }: CondensedButtonProps): ReactElement {
    return (
        <Button type={type} onClick={onClick} colorType="base" isCondensed>
            {children}
        </Button>
    );
}

export default CondensedButton;
