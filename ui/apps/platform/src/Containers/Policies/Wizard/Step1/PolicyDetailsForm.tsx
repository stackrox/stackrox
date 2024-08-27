import React, { ReactElement } from 'react';
import { Alert, Divider, Flex, Title } from '@patternfly/react-core';

import MitreAttackVectorsViewContainer from 'Containers/MitreAttackVectors/MitreAttackVectorsViewContainer';

import PolicyMetadataFormSection from './PolicyMetadataFormSection';
import MitreAttackVectorsFormSection from './MitreAttackVectorsFormSection';

import './PolicyDetailsForm.css';

type PolicyDetailsFormProps = {
    id: string;
    mitreVectorsLocked: boolean;
};

function PolicyDetailsForm({ id, mitreVectorsLocked }: PolicyDetailsFormProps): ReactElement {
    return (
        <Flex
            direction={{ default: 'column' }}
            spaceItems={{ default: 'spaceItemsNone' }}
            flexWrap={{ default: 'nowrap' }}
        >
            <Flex
                direction={{ default: 'column' }}
                spaceItems={{ default: 'spaceItemsSm' }}
                className="pf-v5-u-p-lg"
            >
                <Title headingLevel="h2">Details</Title>
                <div>Describe general information about your policy.</div>
            </Flex>
            <Divider component="div" />
            <div className="pf-v5-u-p-lg">
                <PolicyMetadataFormSection />
            </div>
            <Divider component="div" />
            <Flex
                direction={{ default: 'column' }}
                spaceItems={{ default: 'spaceItemsSm' }}
                className="pf-v5-u-p-lg pf-v5-u-pb-0"
            >
                <Title headingLevel="h2">MITRE ATT&amp;CK</Title>
                <div>
                    MITRE ATT&CK is a globally-accessible knowledge base of adversary tactics and
                    techniques based on real-world observations. The ATT&CK knowledge base is used
                    as a foundation for the development of specific threat models and methodologies
                    in the private sector, in government, and in the cybersecurity product and
                    service community.
                </div>
                {mitreVectorsLocked ? (
                    <>
                        <Alert
                            variant="info"
                            isInline
                            title="Editing MITRE ATT&CK is disabled for system default policies"
                            component="p"
                            className="pf-v5-u-mt-sm"
                        >
                            If you need to edit MITRE ATT&CK, clone this policy or create a new
                            policy.
                        </Alert>
                        <MitreAttackVectorsViewContainer policyId={id} />
                    </>
                ) : (
                    <MitreAttackVectorsFormSection />
                )}
            </Flex>
        </Flex>
    );
}

export default PolicyDetailsForm;
