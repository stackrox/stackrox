import React from 'react';
import { Spinner } from '@patternfly/react-core';

import { TbodyFullCentered } from './TbodyFullCentered';

export type TbodyLoadingProps = {
    colSpan: number;
};

export function TbodyLoading({ colSpan }: TbodyLoadingProps) {
    return (
        <TbodyFullCentered colSpan={colSpan}>
            <Spinner aria-label="Loading table data" />
        </TbodyFullCentered>
    );
}
