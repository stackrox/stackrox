import React, { useState } from 'react';
import { LabelGroup, Label, Tooltip } from '@patternfly/react-core';
import { CopyIcon } from '@patternfly/react-icons';
import { gql } from '@apollo/client';

import { getDistanceStrictAsPhrase, getDateTime } from 'utils/dateUtils';

export type ImageDetails = {
    deploymentCount: number;
    operatingSystem: string;
    metadata: {
        v1: {
            created: Date | null;
            digest: string;
        } | null;
    } | null;
    dataSource: { id: string; name: string } | null;
    scanTime: Date | null;
};

export const imageDetailsFragment = gql`
    fragment ImageDetails on Image {
        id
        deploymentCount
        operatingSystem
        metadata {
            v1 {
                created
                digest
            }
        }
        dataSource {
            id
            name
        }
        scanTime
    }
`;

export type ImageDetailBadgesProps = {
    imageData: ImageDetails;
};

function ImageDetailBadges({ imageData }: ImageDetailBadgesProps) {
    const [hasSuccessfulCopy, setHasSuccessfulCopy] = useState(false);

    const { deploymentCount, operatingSystem, metadata, dataSource, scanTime } = imageData;
    const created = metadata?.v1?.created;
    const sha = metadata?.v1?.digest;
    const isActive = deploymentCount > 0;

    function copyToClipboard(imageSha: string) {
        navigator.clipboard
            .writeText(imageSha)
            .then(() => setHasSuccessfulCopy(true))
            .catch(() => {
                // Permission is not required to write to the clipboard in secure contexts when initiated
                // via a user event so this Promise should not reject
            })
            .finally(() => {
                setTimeout(() => setHasSuccessfulCopy(false), 2000);
            });
    }

    return (
        <LabelGroup numLabels={Infinity}>
            <Label isCompact color={isActive ? 'green' : 'gold'}>
                {isActive ? 'Active' : 'Inactive'}
            </Label>
            {operatingSystem && <Label isCompact>OS: {operatingSystem}</Label>}
            {created && (
                <Label isCompact>Age: {getDistanceStrictAsPhrase(created, new Date())}</Label>
            )}
            {scanTime && (
                <Label isCompact>
                    Scan time: {getDateTime(scanTime)} by {dataSource?.name ?? 'Unknown Scanner'}
                </Label>
            )}
            {sha && (
                <Tooltip content="Copy image SHA to clipboard">
                    <Label
                        style={{ cursor: 'pointer' }}
                        icon={<CopyIcon />}
                        isCompact
                        color={hasSuccessfulCopy ? 'green' : 'grey'}
                        onClick={() => copyToClipboard(sha)}
                    >
                        {hasSuccessfulCopy ? 'Copied!' : 'SHA'}
                    </Label>
                </Tooltip>
            )}
        </LabelGroup>
    );
}

export default ImageDetailBadges;
