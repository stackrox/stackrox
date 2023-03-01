import React, { ReactElement } from 'react';
import { Label } from '@patternfly/react-core';
import { Traits } from 'types/traits.proto';
import { originLabelColours, traitsOriginLabels } from './traits';

export type TraitsOriginLabelProps = {
    traits?: Traits;
};

export function TraitsOriginLabel({ traits }: TraitsOriginLabelProps): ReactElement {
    const originLabel =
        traits && traits.origin && traitsOriginLabels[traits.origin]
            ? traitsOriginLabels[traits.origin]
            : 'User';
    return <Label color={originLabelColours[originLabel]}>{originLabel}</Label>;
}
