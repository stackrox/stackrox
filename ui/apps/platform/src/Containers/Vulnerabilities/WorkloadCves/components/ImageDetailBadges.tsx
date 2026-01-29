import { Label, LabelGroup } from '@patternfly/react-core';
import { gql } from '@apollo/client';

import { getDateTime, getDistanceStrict } from 'utils/dateUtils';
import type { SignatureVerificationResult } from '../../types';
import SignatureCountLabel from './SignatureCountLabel';
import VerifiedSignatureLabel, { getVerifiedSignatureInResults } from './VerifiedSignatureLabel';

export type BaseImage = {
    imageSha: string;
    names: string[];
    created?: string;
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
    baseImage: BaseImage | null;
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
        baseImage {
            imageSha
            names
            created
        }
    }
`;

export const imageV2DetailsFragment = gql`
    fragment ImageV2Details on ImageV2 {
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
        baseImage {
            imageSha
            names
            created
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
