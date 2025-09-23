import React from 'react';
import { Label, LabelProps } from '@patternfly/react-core';

export type TechPreviewLabelProps = LabelProps;

function TechPreviewLabel({ className, ...props }: TechPreviewLabelProps) {
    return (
        <Label
            isCompact
            color="orange"
            className={className}
            {...props}
        >
            Tech preview
        </Label>
    );
}

export default TechPreviewLabel;
