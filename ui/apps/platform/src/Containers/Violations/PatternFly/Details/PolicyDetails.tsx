import React from 'react';
import { Flex } from '@patternfly/react-core';

import Fields from 'Containers/Policies/Wizard/Details/Fields';
import ConfigurationFields from 'Containers/Policies/Wizard/Details/ConfigurationFields';
import BooleanPolicySection from 'Containers/Policies/Wizard/Form/BooleanPolicySection';

function PolicyDetails({ policy }) {
    // If the policy version is not set, that means this is a legacy policy.
    // Legacy policies are only displayed when we display old alerts.
    const isLegacyPolicy = !policy.policyVersion;

    return (
        <Flex>
            <Flex flex={{ default: 'flex_1' }}>
                <Fields policy={policy} />
            </Flex>
            <Flex flex={{ default: 'flex_1' }}>
                {!isLegacyPolicy && <BooleanPolicySection readOnly initialValues={policy} />}
                {isLegacyPolicy && <ConfigurationFields policy={policy} />}
            </Flex>
        </Flex>
    );
}

export default PolicyDetails;
