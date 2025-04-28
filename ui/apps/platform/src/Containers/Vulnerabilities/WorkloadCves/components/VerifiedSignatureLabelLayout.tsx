import React from 'react';
import { Button, Label, Popover } from '@patternfly/react-core';
import { CheckSquareIcon } from '@patternfly/react-icons';

import PopoverBodyContent from 'Components/PopoverBodyContent';
import { SignatureVerificationResult } from '../../types';

export type VerifiedSignatureLabelProps = {
    results: SignatureVerificationResult[];
};

function VerifiedSignatureLabel({ results }: VerifiedSignatureLabelProps) {
    return (
        <>
            <Popover
                aria-label="Verified image references"
                bodyContent={
                    <PopoverBodyContent
                        headerContent="Verified image references"
                        bodyContent={
                            <>
                                {results.map((result) => (
                                    <>
                                        <strong>{result.verifierId}:</strong>
                                        <ul>
                                            {result.verifiedImageReferences.map((name) => (
                                                <li>{name}</li>
                                            ))}
                                        </ul>
                                    </>
                                ))}
                            </>
                        }
                    />
                }
                enableFlip
                position="top"
            >
                <Button variant="plain" className="pf-v5-u-p-0">
                    <Label
                        isCompact
                        variant="outline"
                        color="green"
                        className="pf-v5-u-mt-xs"
                        icon={<CheckSquareIcon />}
                    >
                        Verified signature
                    </Label>
                </Button>
            </Popover>
        </>
    );
}

export default VerifiedSignatureLabel;
