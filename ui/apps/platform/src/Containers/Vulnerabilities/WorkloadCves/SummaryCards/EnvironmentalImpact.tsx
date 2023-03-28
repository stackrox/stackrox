import React from 'react';
import { Card, CardTitle, CardBody } from '@patternfly/react-core';

export type EnvironmentalImpactProps = {
    className?: string;
    affectedImageCount: number;
    totalImagesCount: number;
};

function EnvironmentalImpact({
    className = '',
    affectedImageCount,
    totalImagesCount,
}: EnvironmentalImpactProps) {
    return (
        <Card className={className} isCompact>
            <CardTitle>Environmental impact</CardTitle>
            <CardBody>
                {affectedImageCount}/{totalImagesCount} images affected
            </CardBody>
        </Card>
    );
}

export default EnvironmentalImpact;
