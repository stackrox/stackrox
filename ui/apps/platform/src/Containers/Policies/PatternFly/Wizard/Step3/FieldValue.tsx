import React from 'react';
import { Flex, Button, Divider } from '@patternfly/react-core';
import { TimesIcon } from '@patternfly/react-icons';

import { Descriptor } from 'Containers/Policies/Wizard/Form/descriptors';
import Field from './Field';

type FieldValueProps = {
    name: string;
    length: number;
    descriptor: Descriptor;
    readOnly?: boolean;
    handleRemoveValue: () => void;
};

function FieldValue({
    name,
    length,
    handleRemoveValue,
    descriptor,
    readOnly = false,
}: FieldValueProps) {
    return (
        <div data-testid="policy-field-value">
            <Flex
                direction={{ default: 'row' }}
                flexWrap={{ default: 'nowrap' }}
                alignItems={{ default: 'alignItemsStretch' }}
            >
                <Field descriptor={descriptor} name={name} readOnly={readOnly} />
                {/* only show remove button if there is more than one value */}
                {!readOnly && length > 1 && (
                    <Button onClick={handleRemoveValue} variant="tertiary">
                        <TimesIcon />
                    </Button>
                )}
            </Flex>
        </div>
    );
}

export default FieldValue;
