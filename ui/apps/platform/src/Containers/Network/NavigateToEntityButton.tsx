import React, { ReactElement } from 'react';
import { ArrowUpRight } from 'react-feather';

import { EntityType } from 'Containers/Network/networkTypes';

import { CondensedButton } from '@stackrox/ui-components';

export type NavigateToEntityButtonProps = {
    entityId: string;
    entityType: EntityType;
    onClick: (entityId: string, entityType: EntityType) => void;
};

function NavigateToEntityButton({
    entityId,
    entityType,
    onClick,
}: NavigateToEntityButtonProps): ReactElement {
    function onClickHandler(): void {
        onClick(entityId, entityType);
    }
    return (
        <CondensedButton type="button" onClick={onClickHandler}>
            <ArrowUpRight className="h-3 w-3 mr-1" />
            Navigate
        </CondensedButton>
    );
}

export default NavigateToEntityButton;
