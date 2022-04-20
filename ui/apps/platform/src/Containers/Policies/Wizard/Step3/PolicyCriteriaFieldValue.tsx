import React from 'react';
import { Flex, Button } from '@patternfly/react-core';
import { TimesIcon } from '@patternfly/react-icons';

import { Descriptor } from './policyCriteriaDescriptors';
import PolicyCriteriaFieldInput from './PolicyCriteriaFieldInput';

type FieldValueProps = {
    name: string;
    length: number;
    descriptor: Descriptor;
    readOnly?: boolean;
    handleRemoveValue: () => void;
};

function PolicyCriteriaFieldValue({
    name,
    length,
    handleRemoveValue,
    descriptor,
    readOnly = false,
}: FieldValueProps) {
    return (
        <div data-testid="policy-criteria-value">
            <Flex
                direction={{ default: 'row' }}
                flexWrap={{ default: 'nowrap' }}
                alignItems={{ default: 'alignItemsStretch' }}
            >
                <PolicyCriteriaFieldInput descriptor={descriptor} name={name} readOnly={readOnly} />
                {/* only show remove button if there is more than one value */}
                {!readOnly && length > 1 && (
                    <Button
                        onClick={handleRemoveValue}
                        variant="tertiary"
                        data-testid="delete-policy-criteria-value-btn"
                    >
                        <TimesIcon />
                    </Button>
                )}
            </Flex>
        </div>
    );
}

export default PolicyCriteriaFieldValue;
