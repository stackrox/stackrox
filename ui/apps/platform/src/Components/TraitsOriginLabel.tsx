import type { ReactElement } from 'react';
import { Label } from '@patternfly/react-core';
import type { Traits } from 'types/traits.proto';
import { getOriginLabel, originLabelColours } from 'utils/traits.utils';

export type TraitsOriginLabelProps = {
    traits?: Traits;
};

function TraitsOriginLabel({ traits }: TraitsOriginLabelProps): ReactElement {
    const originLabel = getOriginLabel(traits);
    const color = originLabelColours[originLabel];
    return <Label color={color}>{originLabel}</Label>;
}

export default TraitsOriginLabel;
