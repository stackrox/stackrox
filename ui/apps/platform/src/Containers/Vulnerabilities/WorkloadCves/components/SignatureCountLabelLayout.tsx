import React from 'react';
import { Button, Flex, FlexItem, Label, Popover, Text } from '@patternfly/react-core';

import useMetadata from 'hooks/useMetadata';
import { getProductBranding } from 'constants/productBranding';
import { getVersionedDocs } from 'utils/versioning';
import ExternalLink from 'Components/PatternFly/IconText/ExternalLink';
import PopoverBodyContent from 'Components/PopoverBodyContent';

export type SignatureCountLabelProps = {
    count: number;
};

const noSignatureMessage = 'No signature found';
const { shortName } = getProductBranding();

function getBodyContent(version: string) {
    return (
        <Flex direction={{ default: 'column' }}>
            <FlexItem>
                <Text>
                    Image signatures increase the security and transparency of container images.
                </Text>
            </FlexItem>
            <FlexItem>
                <Text>
                    Create at least one image signature integration to download and verify image
                    signatures.
                </Text>
            </FlexItem>
            <FlexItem>
                <Text>
                    For more information, see{' '}
                    <ExternalLink>
                        <a
                            href={getVersionedDocs(version, 'operating/verify-image-signatures')}
                            target="_blank"
                            rel="noopener noreferrer"
                        >
                            {shortName} documentation
                        </a>
                    </ExternalLink>
                </Text>
            </FlexItem>
        </Flex>
    );
}

function SignatureCountLabel({ count }: SignatureCountLabelProps) {
    const { version } = useMetadata();

    if (count === 0) {
        return (
            <Popover
                aria-label={noSignatureMessage}
                bodyContent={
                    <PopoverBodyContent
                        headerContent={noSignatureMessage}
                        bodyContent={getBodyContent(version)}
                    />
                }
                enableFlip
                hasAutoWidth
                position="top"
            >
                <Button variant="plain" className="pf-v5-u-p-0">
                    <Label color="gold">{noSignatureMessage}</Label>
                </Button>
            </Popover>
        );
    }

    const signatureMessage = `Signatures: ${count}`;
    return (
        <Popover
            aria-label="Signature count"
            bodyContent={
                <PopoverBodyContent
                    headerContent={signatureMessage}
                    bodyContent={getBodyContent(version)}
                />
            }
            enableFlip
            hasAutoWidth
            position="top"
        >
            <Button variant="plain" className="pf-v5-u-p-0">
                <Label>{signatureMessage}</Label>
            </Button>
        </Popover>
    );
}

export default SignatureCountLabel;
