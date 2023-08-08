import React, { ReactElement } from 'react';
import { Alert, Divider, Flex, FlexItem, Title } from '@patternfly/react-core';

import MitreAttackVectorsViewContainer from 'Containers/MitreAttackVectors/MitreAttackVectorsViewContainer';

import PolicyMetadataFormSection from './PolicyMetadataFormSection';
import AttachNotifiersFormSection from './AttachNotifiersFormSection';
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
                className="pf-u-p-lg"
            >
                <Title headingLevel="h2">Policy details</Title>
                <div>Describe general information about your policy.</div>
            </Flex>
            <Divider component="div" />
            <Flex>
                <Flex
                    spaceItems={{ default: 'spaceItemsNone' }}
                    flexWrap={{ default: 'wrap', md: 'nowrap' }}
                    fullWidth={{ default: 'fullWidth' }}
                >
                    <FlexItem
                        className="pf-u-p-lg"
                        grow={{ default: 'grow' }}
                        flex={{ default: 'flex_1' }}
                    >
                        <PolicyMetadataFormSection />
                    </FlexItem>
                    <Divider component="div" isVertical />
                    <FlexItem className="attach-notifiers-section">
                        <AttachNotifiersFormSection />
                    </FlexItem>
                </Flex>
                <Divider component="div" />
                <Flex
                    direction={{ default: 'column' }}
                    spaceItems={{ default: 'spaceItemsSm' }}
                    className="pf-u-p-lg pf-u-pb-0"
                >
                    <Title headingLevel="h2">MITRE ATT&amp;CK</Title>
                    <div>
                        MITRE ATT&CK is a globally-accessible knowledge base of adversary tactics
                        and techniques based on real-world observations. The ATT&CK knowledge base
                        is used as a foundation for the development of specific threat models and
                        methodologies in the private sector, in government, and in the cybersecurity
                        product and service community.
                    </div>
                    {mitreVectorsLocked ? (
                        <>
                            <Alert
                                variant="info"
                                isInline
                                title="Editing MITRE ATT&CK is disabled for system default policies"
                                component="h3"
                                className="pf-u-mt-sm"
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
        </Flex>
    );
}

export default PolicyDetailsForm;
