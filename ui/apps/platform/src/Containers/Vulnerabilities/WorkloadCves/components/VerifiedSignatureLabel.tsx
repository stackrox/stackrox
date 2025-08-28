import React, { CSSProperties } from 'react';
import { Button, Flex, FlexItem, Label, List, ListItem, Popover } from '@patternfly/react-core';
import { CheckCircleIcon } from '@patternfly/react-icons';

import PopoverBodyContent from 'Components/PopoverBodyContent';
import { SignatureVerificationResult } from '../../types';

export function getVerifiedSignatureInResults(
    results: SignatureVerificationResult[] | null | undefined
): SignatureVerificationResult[] {
    const verifiedSignatureResults = results?.filter((result) => result.status === 'VERIFIED');
    return Array.isArray(verifiedSignatureResults) && verifiedSignatureResults.length !== 0
        ? verifiedSignatureResults
        : [];
}

export type VerifiedSignatureLabelProps = {
    verifiedSignatureResults: SignatureVerificationResult[];
    className?: string;
    isCompact?: boolean;
    variant?: 'outline' | 'filled';
};

// Separate list from the title with same margin-top as second list item from the first.
const styleList = {
    marginTop: 'var(--pf-v5-c-list--li--MarginTop)',
} as CSSProperties;

function VerifiedSignatureLabel({
    verifiedSignatureResults,
    className,
    isCompact,
    variant,
}: VerifiedSignatureLabelProps) {
    return (
        <Popover
            aria-label="Verified image references"
            bodyContent={
                <PopoverBodyContent
                    headerContent="Verified image references"
                    bodyContent={
                        <Flex
                            direction={{ default: 'column' }}
                            spaceItems={{ default: 'spaceItemsMd' }}
                        >
                            {verifiedSignatureResults?.map((result) => (
                                <FlexItem key={result.verifierId}>
                                    <strong>{result.verifierId}</strong>
                                    <List style={styleList}>
                                        {result.verifiedImageReferences?.map((name) => (
                                            <ListItem key={name}>{name}</ListItem>
                                        ))}
                                    </List>
                                </FlexItem>
                            ))}
                        </Flex>
                    }
                />
            }
            enableFlip
            hasAutoWidth
            position="top"
        >
            <Button variant="plain">
                <Label
                    isCompact={isCompact}
                    variant={variant}
                    color="green"
                    className={className}
                    icon={<CheckCircleIcon />}
                >
                    Verified signature
                </Label>
            </Button>
        </Popover>
    );
}

export default VerifiedSignatureLabel;
