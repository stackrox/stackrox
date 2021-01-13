import React, { ReactElement } from 'react';

import Button, { HOCButtonProps } from '../Button';

export type CondensedAlertButtonProps = HOCButtonProps;

function CondensedAlertButton({
    type,
    onClick,
    children,
    isDisabled,
}: CondensedAlertButtonProps): ReactElement {
    return (
        <Button type={type} onClick={onClick} colorType="alert" isDisabled={isDisabled} isCondensed>
            {children}
        </Button>
    );
}

export default CondensedAlertButton;
