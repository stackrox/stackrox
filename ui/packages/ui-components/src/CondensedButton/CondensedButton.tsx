import React, { ReactElement } from 'react';

import Button, { HOCButtonProps } from '../Button';

export type CondensedButtonProps = HOCButtonProps;

function CondensedButton({
    type,
    onClick,
    children,
    isDisabled,
}: CondensedButtonProps): ReactElement {
    return (
        <Button type={type} onClick={onClick} colorType="base" isDisabled={isDisabled} isCondensed>
            {children}
        </Button>
    );
}

export default CondensedButton;
