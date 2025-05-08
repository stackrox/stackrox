import React, { CSSProperties } from 'react';
import { Button, Flex, FlexItem, Label, List, ListItem, Popover } from '@patternfly/react-core';
import { CheckCircleIcon } from '@patternfly/react-icons';

import PopoverBodyContent from 'Components/PopoverBodyContent';
import { SignatureVerificationResult } from '../../types';

export type VerifiedSignatureLabelProps = {
    results: SignatureVerificationResult[];
};

function VerifiedSignatureLabel({ results }: VerifiedSignatureLabelProps) {
    // Separate list from the title with same margin-top as second list item from the first.
    const styleList = {
        marginTop: 'var(--pf-v5-c-list--li--MarginTop)',
    } as CSSProperties;

    return (
        <>
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
                                {results.map((result) => (
                                    <FlexItem key={result.verifierId}>
                                        <strong>{result.verifierId}</strong>
                                        <List style={styleList}>
                                            {result.verifiedImageReferences.map((name) => (
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
                <Button variant="plain" className="pf-v5-u-p-0">
                    <Label
                        isCompact
                        variant="outline"
                        color="green"
                        className="pf-v5-u-mt-xs"
                        icon={<CheckCircleIcon />}
                    >
                        Verified signature
                    </Label>
                </Button>
            </Popover>
        </>
    );
}

export default VerifiedSignatureLabel;
