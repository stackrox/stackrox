import type { CSSProperties } from 'react';
import {
    ClipboardCopy,
    Flex,
    FlexItem,
    Label,
    List,
    ListItem,
    Popover,
} from '@patternfly/react-core';
import { CheckCircleIcon } from '@patternfly/react-icons';

import PopoverBodyContent from 'Components/PopoverBodyContent';
import type { SignatureVerificationResult } from '../../types';
import SignatureIntegrationLink from './SignatureIntegrationLink';

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

const clipboardCopyMaxWidthStyle = {
    '--pf-v5-u-max-width--MaxWidth': '64ch',
} as CSSProperties;

function VerifiedSignatureLabel({
    verifiedSignatureResults,
    className,
    isCompact,
    variant,
}: VerifiedSignatureLabelProps) {
    // TODO replace style={{ cursor: 'pointer' }} prop with isClickable prop in PatternFly 6?
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
                                    <strong>
                                        <SignatureIntegrationLink result={result} />
                                    </strong>
                                    <List style={styleList}>
                                        {result.verifiedImageReferences?.map((name) => (
                                            <ListItem key={name}>
                                                <ClipboardCopy
                                                    className="pf-v5-u-max-width pf-v5-u-display-inline-flex pf-v5-u-align-items-center"
                                                    style={clipboardCopyMaxWidthStyle}
                                                    clickTip="Copied!"
                                                    hoverTip="Copy"
                                                    variant="inline-compact"
                                                >
                                                    {name}
                                                </ClipboardCopy>
                                            </ListItem>
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
            <Label
                isCompact={isCompact}
                variant={variant}
                color="green"
                className={className}
                icon={<CheckCircleIcon />}
                style={{ cursor: 'pointer' }}
            >
                Verified signature
            </Label>
        </Popover>
    );
}

export default VerifiedSignatureLabel;
