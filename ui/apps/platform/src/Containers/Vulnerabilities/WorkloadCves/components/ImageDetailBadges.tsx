import { Label, LabelGroup } from '@patternfly/react-core';
import { gql } from '@apollo/client';

import { getDateTime, getDistanceStrict } from 'utils/dateUtils';
import type { SignatureVerificationResult } from '../../types';
import SignatureCountLabel from './SignatureCountLabel';
import VerifiedSignatureLabel, { getVerifiedSignatureInResults } from './VerifiedSignatureLabel';

export type BaseImageInfo = {
    baseImageId: string;
    baseImageFullName: string;
    baseImageDigest: string;
    baseImageCreated?: string;
};

export type ImageDetails = {
    deploymentCount: number;
    operatingSystem: string;
    metadata: {
        v1: {
            created: string | null;
        } | null;
    } | null;
    dataSource: { name: string } | null;
    scanTime: string | null;
    scanNotes: string[];
    notes: string[];
    signatureCount: number;
    signatureVerificationData: {
        results: SignatureVerificationResult[];
    } | null;
    baseImageInfo: BaseImageInfo[];
};

export const imageDetailsFragment = gql`
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
        scanNotes
        notes
        signatureCount
        signatureVerificationData {
            results {
                description
                status
                verificationTime
                verifiedImageReferences
                verifierId
            }
        }
        baseImageInfo {
            baseImageId
            baseImageFullName
            baseImageDigest
            # TODO: Uncomment when backend adds 'baseImageCreated' field to BaseImageInfo GraphQL type
            # baseImageCreated
        }
    }
`;

export const imageV2DetailsFragment = gql`
    fragment ImageDetails on ImageV2 {
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
        scanNotes
        notes
        signatureCount
        signatureVerificationData {
            results {
                description
                status
                verificationTime
                verifiedImageReferences
                verifierId
            }
        }
    }
`;

export type ImageDetailBadgesProps = {
    imageData: ImageDetails;
};

function ImageDetailBadges({ imageData }: ImageDetailBadgesProps) {
    const {
        deploymentCount,
        operatingSystem,
        metadata,
        dataSource,
        scanTime,
        signatureCount,
        signatureVerificationData,
    } = imageData;
    const created = metadata?.v1?.created;
    const isActive = deploymentCount > 0;
    const verifiedSignatureResults = getVerifiedSignatureInResults(
        signatureVerificationData?.results
    );

    return (
        <LabelGroup numLabels={Infinity}>
            <Label color={isActive ? 'green' : 'gold'}>{isActive ? 'Active' : 'Inactive'}</Label>
            {verifiedSignatureResults.length !== 0 && (
                <VerifiedSignatureLabel verifiedSignatureResults={verifiedSignatureResults} />
            )}
            <SignatureCountLabel count={signatureCount} />
            {operatingSystem && <Label>OS: {operatingSystem}</Label>}
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
