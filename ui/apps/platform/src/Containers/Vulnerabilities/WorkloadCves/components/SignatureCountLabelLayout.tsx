import React from 'react';
import { Label } from '@patternfly/react-core';

export type SignatureCountLabelProps = {
    count: number;
};

function SignatureCountLabel({ count }: SignatureCountLabelProps) {
    return (
        <>
            {count === 0 && <Label color="gold">No signature found</Label>}
            {count > 0 && <Label>Signatures: {count}</Label>}
        </>
    );
}

export default SignatureCountLabel;
