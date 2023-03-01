import React from 'react';
import { LabelGroup, Label } from '@patternfly/react-core';

import { getDistanceStrict, getDateTime } from 'utils/dateUtils';
import { graphql } from 'generated/graphql-codegen';
import { ImageDetailsFragment } from 'generated/graphql-codegen/graphql';

export const imageDetailsFragment = graphql(/* GraphQL */ `
    fragment ImageDetails on Image {
        deploymentCount
        operatingSystem
        metadata {
            v1 {
                created
            }
        }
        dataSource {
            name
        }
        scanTime
    }
`);

export type ImageDetailBadgesProps = {
    imageData: ImageDetailsFragment;
};

function ImageDetailBadges({ imageData }: ImageDetailBadgesProps) {
    const { deploymentCount, operatingSystem, metadata, dataSource, scanTime } = imageData;
    const created = metadata?.v1?.created;
    const isActive = deploymentCount > 0;

    return (
        <LabelGroup numLabels={Infinity}>
            <Label color={isActive ? 'green' : 'gold'}>{isActive ? 'Active' : 'Inactive'}</Label>
            <Label>OS: {operatingSystem}</Label>
            {created && <Label>Age: {getDistanceStrict(created, new Date())}</Label>}
            {scanTime && (
                <Label>
                    Scan time: {getDateTime(scanTime)} by {dataSource?.name ?? 'Unknown Scanner'}
                </Label>
            )}
        </LabelGroup>
    );
}

export default ImageDetailBadges;
