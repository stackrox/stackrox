import React from 'react';
import { Card, CardBody } from '@patternfly/react-core';

import useURLStringUnion from 'hooks/useURLStringUnion';
import BaseImageTabToggleGroup from './BaseImageTabToggleGroup';
import BaseImageCVEsTab from '../tabs/BaseImageCVEsTab';
import BaseImageImagesTab from '../tabs/BaseImageImagesTab';

const baseImageTabValues = ['cves', 'images'] as const;

type BaseImageDetailTabsProps = {
    baseImageId: string;
};

function BaseImageDetailTabs({ baseImageId }: BaseImageDetailTabsProps) {
    const [activeTabKey] = useURLStringUnion('tab', baseImageTabValues);

    return (
        <Card>
            <CardBody>
                <div className="pf-v5-u-mb-md">
                    <BaseImageTabToggleGroup />
                </div>
                {activeTabKey === 'cves' && <BaseImageCVEsTab baseImageId={baseImageId} />}
                {activeTabKey === 'images' && <BaseImageImagesTab baseImageId={baseImageId} />}
            </CardBody>
        </Card>
    );
}

export default BaseImageDetailTabs;
