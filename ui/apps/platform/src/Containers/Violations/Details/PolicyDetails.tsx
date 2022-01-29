import React, { ReactElement } from 'react';
import { Flex, FlexItem, Title } from '@patternfly/react-core';

import { Policy } from 'Containers/Violations/types/violationTypes';

import Fields from 'Containers/Policies/Wizard/Details/Fields';
import ConfigurationFields from 'Containers/Policies/Wizard/Details/ConfigurationFields';
import BooleanPolicySection from 'Containers/Policies/Wizard/Form/BooleanPolicySection';
import MitreAttackVectors from 'Containers/MitreAttackVectors/MitreAttackVectorsView';

type PolicyDetailsProps = {
    policy: Policy;
};

function PolicyDetails({ policy }: PolicyDetailsProps): ReactElement {
    // If the policy version is not set, that means this is a legacy policy.
    // Legacy policies are only displayed when we display old alerts.
    const isLegacyPolicy = !policy.policyVersion;

    return (
        <Flex>
            <Flex
                direction={{ default: 'column' }}
                flex={{ default: 'flex_1' }}
                spaceItems={{ default: 'spaceItemsXl' }}
            >
                <FlexItem>
                    <Fields policy={policy} />
                </FlexItem>
                <FlexItem>
                    <Title headingLevel="h3">MITRE ATT&CK</Title>
                    <div className="pf-u-mt-md">
                        {!!policy.id && <MitreAttackVectors policyId={policy.id} />}
                    </div>
                </FlexItem>
            </Flex>
            <Flex flex={{ default: 'flex_1' }}>
                {!isLegacyPolicy && <BooleanPolicySection readOnly initialValues={policy} />}
                {isLegacyPolicy && <ConfigurationFields policy={policy} />}
            </Flex>
        </Flex>
    );
}

export default PolicyDetails;
