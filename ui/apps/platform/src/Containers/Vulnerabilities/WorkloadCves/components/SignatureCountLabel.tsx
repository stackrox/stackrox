import React from 'react';
import { Flex, FlexItem, Label, Popover, Text } from '@patternfly/react-core';

import useMetadata from 'hooks/useMetadata';
import { getProductBranding } from 'constants/productBranding';
import { getVersionedDocs } from 'utils/versioning';
import ExternalLink from 'Components/PatternFly/IconText/ExternalLink';
import PopoverBodyContent from 'Components/PopoverBodyContent';

export type SignatureCountLabelProps = {
    count: number;
};

const noSignatureMessage = 'No signature found';

function getAriaLabel(count: number): string {
    if (count === 0) {
        return noSignatureMessage;
    }
    return 'Signature count';
}

function getHeaderContent(count: number): string {
    if (count === 0) {
        return noSignatureMessage;
    }
    return `Signatures: ${count}`;
}

function getMessage(count: number): string {
    if (count === 0) {
        return noSignatureMessage;
    }
    return `Signatures: ${count}`;
}

function getColor(count: number): 'gold' | undefined {
    if (count === 0) {
        return 'gold';
    }
    return undefined;
}

function SignatureCountLabel({ count }: SignatureCountLabelProps) {
    const { shortName } = getProductBranding();
    const { version } = useMetadata();
    return (
        <Popover
            aria-label={getAriaLabel(count)}
            bodyContent={
                <PopoverBodyContent
                    headerContent={getHeaderContent(count)}
                    bodyContent={
                        <Flex direction={{ default: 'column' }}>
                            <FlexItem>
                                <Text>
                                    Image signatures increase the security and transparency of
                                    container images.
                                </Text>
                            </FlexItem>
                            <FlexItem>
                                <Text>
                                    Create at least one image signature integration to download and
                                    verify image signatures.
                                </Text>
                            </FlexItem>
                            <FlexItem>
                                <Text>
                                    For more information, see{' '}
                                    <ExternalLink>
                                        <a
                                            href={getVersionedDocs(
                                                version,
                                                'operating/verify-image-signatures'
                                            )}
                                            target="_blank"
                                            rel="noopener noreferrer"
                                        >
                                            {shortName} documentation
                                        </a>
                                    </ExternalLink>
                                </Text>
                            </FlexItem>
                        </Flex>
                    }
                />
            }
            enableFlip
            hasAutoWidth
            position="top"
        >
            <Label color={getColor(count)} style={{ cursor: 'pointer' }}>
                {getMessage(count)}
            </Label>
        </Popover>
    );
}

export default SignatureCountLabel;
