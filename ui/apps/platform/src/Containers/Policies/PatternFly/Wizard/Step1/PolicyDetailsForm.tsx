import React, { ReactElement } from 'react';
import { Alert, Divider, Flex, FlexItem, Title } from '@patternfly/react-core';

import MitreAttackVectorsView from 'Containers/MitreAttackVectors/MitreAttackVectorsView';

import PolicyMetadataFormSection from './PolicyMetadataFormSection';
import AttachNotifiersFormSection from './AttachNotifiersFormSection';
import MitreAttackVectorsFormSection from './MitreAttackVectorsFormSection';

type PolicyDetailsFormProps = {
    id: string;
    mitreVectorsLocked: boolean;
};

function PolicyDetailsForm({ id, mitreVectorsLocked }: PolicyDetailsFormProps): ReactElement {
    return (
        <div>
            <Title headingLevel="h2">Policy details</Title>
            <div className="pf-u-mb-md pf-u-mt-sm">
                Describe general information about your policy.
            </div>
            <Divider component="div" />
            <Flex direction={{ default: 'row' }}>
                <FlexItem className="pf-u-mb-md" grow={{ default: 'grow' }}>
                    <PolicyMetadataFormSection />
                </FlexItem>
                <Divider component="div" isVertical />
                <FlexItem className="pf-u-w-33">
                    <AttachNotifiersFormSection />
                </FlexItem>
            </Flex>
            <Divider component="div" className="pf-u-pb-md" />
            <Title headingLevel="h2">MITRE ATT&amp;CK</Title>
            <div className="pf-u-pt-sm">
                MITRE ATT&CK is a globally-accessible knowledge base of adversary tactics and
                techniques based on real-world observations. The ATT&CK knowledge base is used as a
                foundation for the development of specific threat models and methodologies in the
                private sector, in government, and in the cybersecurity product and service
                community.
            </div>
            {mitreVectorsLocked ? (
                <>
                    <Alert
                        variant="info"
                        isInline
                        title="Editing MITRE ATT&CK is disabled for system default policies"
                        className="pf-u-mt-sm"
                    >
                        If you need to edit MITRE ATT&CK, clone this policy or create a new policy.
                    </Alert>
                    <MitreAttackVectorsView policyId={id} />
                </>
            ) : (
                <MitreAttackVectorsFormSection />
            )}
        </div>
    );
}

export default PolicyDetailsForm;
