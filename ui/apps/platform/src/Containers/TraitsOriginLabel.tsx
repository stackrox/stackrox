import React, { ReactElement } from 'react';
import { Label } from '@patternfly/react-core';
import { Traits } from 'types/traits.proto';
import { getOriginLabel, originLabelColours } from './traits';

export type TraitsOriginLabelProps = {
    traits?: Traits;
};

function TraitsOriginLabel({ traits }: TraitsOriginLabelProps): ReactElement {
    const originLabel = getOriginLabel(traits);
    return <Label color={originLabelColours[originLabel]}>{originLabel}</Label>;
}

export default TraitsOriginLabel;
