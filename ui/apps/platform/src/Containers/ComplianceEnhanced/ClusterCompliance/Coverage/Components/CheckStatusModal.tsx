import React from 'react';
import {
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    Divider,
    Flex,
    Label,
    LabelGroup,
    List,
    ListItem,
    Modal,
    ModalVariant,
    Text,
    Title,
    ToggleGroup,
    ToggleGroupItem,
} from '@patternfly/react-core';

import IconText from 'Components/PatternFly/IconText/IconText';
import { ComplianceCheckResult, ComplianceCheckStatus } from 'services/ComplianceEnhancedService';

import { getClusterResultsStatusObject } from '../compliance.coverage.utils';

import './CheckStatusModal.css';

type CheckStatusModalProps = {
    checkResult: ComplianceCheckResult;
    isOpen: boolean;
    status: ComplianceCheckStatus | null;
    handleClose: () => void;
};

function CheckStatusModal({ checkResult, isOpen, status, handleClose }: CheckStatusModalProps) {
    const { checkName, description, instructions, rationale, warnings } = checkResult;

    const statusObj = status ? getClusterResultsStatusObject(status) : null;

    const header = (
        <Flex direction={{ default: 'column' }}>
            <Title headingLevel="h1">{checkName}</Title>
            {statusObj && (
                <LabelGroup numLabels={1}>
                    <Label>
                        <IconText icon={statusObj.icon} text={statusObj.statusText} />
                    </Label>
                </LabelGroup>
            )}
            <Text>{rationale}</Text>
            <ToggleGroup aria-label="Toggle for check details modal view">
                <ToggleGroupItem
                    text="Check details"
                    buttonId="check-details-toggle-group"
                    isSelected
                />
                <ToggleGroupItem
                    text="Remediation details (coming soon)"
                    buttonId="remediation-details-toggle-group"
                    isDisabled
                />
            </ToggleGroup>
            <Divider component="div" className="pf-u-pb-md" />
        </Flex>
    );

    return (
        <>
            <Modal
                isOpen={isOpen}
                onClose={handleClose}
                variant={ModalVariant.large}
                tabIndex={0}
                header={header}
                aria-label="Check status details modal"
            >
                <DescriptionList>
                    <DescriptionListGroup>
                        <DescriptionListTerm>Description</DescriptionListTerm>
                        <DescriptionListDescription className="formatted-text">
                            {description}
                        </DescriptionListDescription>
                    </DescriptionListGroup>
                    <DescriptionListGroup>
                        <DescriptionListTerm>Instruction</DescriptionListTerm>
                        <DescriptionListDescription className="formatted-text">
                            {instructions}
                        </DescriptionListDescription>
                    </DescriptionListGroup>
                    {warnings.length > 0 && (
                        <DescriptionListGroup>
                            <DescriptionListTerm>Warning(s)</DescriptionListTerm>
                            <DescriptionListDescription>
                                <List>
                                    {warnings.map((warning) => (
                                        <ListItem key={warning}>{warning}</ListItem>
                                    ))}
                                </List>
                            </DescriptionListDescription>
                        </DescriptionListGroup>
                    )}
                </DescriptionList>
            </Modal>
        </>
    );
}

export default CheckStatusModal;
