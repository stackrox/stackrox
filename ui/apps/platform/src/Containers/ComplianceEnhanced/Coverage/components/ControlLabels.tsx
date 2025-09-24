import React from 'react';
import type { ReactElement } from 'react';
import { Label, LabelGroup } from '@patternfly/react-core';

import type { ComplianceControl } from 'services/ComplianceCommon';

type ControlLabelsProps = {
    controls: ComplianceControl[];
    numLabels?: number;
};

function ControlLabels({ controls, numLabels = Infinity }: ControlLabelsProps): ReactElement {
    return (
        <LabelGroup numLabels={numLabels}>
            {controls.map(({ control, standard }) => (
                <Label key={control}>{`${standard} ${control}`}</Label>
            ))}
        </LabelGroup>
    );
}

export default ControlLabels;
