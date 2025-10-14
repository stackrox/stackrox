import React from 'react';
import { Label } from '@patternfly/react-core';

export type LayerType = 'base' | 'application';

type LayerTypeBadgeProps = {
    layerType: LayerType;
    showIcon?: boolean;
};

/**
 * Badge component to indicate whether a CVE/component is from the base image or application layer
 * - Base Image: Blue badge
 * - Application: Green badge
 */
function LayerTypeBadge({ layerType, showIcon = false }: LayerTypeBadgeProps) {
    if (layerType === 'base') {
        return (
            <Label color="blue" icon={showIcon ? undefined : null}>
                Base Image
            </Label>
        );
    }

    return (
        <Label color="green" icon={showIcon ? undefined : null}>
            Application
        </Label>
    );
}

export default LayerTypeBadge;
