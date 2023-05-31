import React from 'react';
import { Card, CardTitle, CardBody } from '@patternfly/react-core';

export type AffectedImagesProps = {
    className?: string;
    affectedImageCount: number;
    totalImagesCount: number;
};

function AffectedImages({
    className = '',
    affectedImageCount,
    totalImagesCount,
}: AffectedImagesProps) {
    return (
        <Card className={className} isCompact isFlat>
            <CardTitle>Affected images</CardTitle>
            <CardBody>
                {affectedImageCount}/{totalImagesCount} images affected
            </CardBody>
        </Card>
    );
}

export default AffectedImages;
