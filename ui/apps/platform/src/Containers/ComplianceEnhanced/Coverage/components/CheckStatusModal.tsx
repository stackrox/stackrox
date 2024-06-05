import React from 'react';
import {
    Button,
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
    Title,
    ToggleGroup,
    ToggleGroupItem,
} from '@patternfly/react-core';

import IconText from 'Components/PatternFly/IconText/IconText';
import { ComplianceCheckResult } from 'services/ComplianceResultsService';

import { getClusterResultsStatusObject } from '../compliance.coverage.utils';

import './CheckStatusModal.css';

type CheckStatusModalProps = {
    checkResult: ComplianceCheckResult;
    clusterName: string;
    isOpen: boolean;
    handleClose: () => void;
};

function CheckStatusModal({
    checkResult,
    clusterName,
    isOpen,
    handleClose,
}: CheckStatusModalProps) {
    const { checkName, description, instructions, rationale, warnings } = checkResult;

    const statusObj = getClusterResultsStatusObject(checkResult.status);

    const header = (
        <Flex direction={{ default: 'column' }}>
            <Title headingLevel="h1">{checkName}</Title>
            {statusObj && (
                <LabelGroup numLabels={2}>
                    <Label color={statusObj.color}>
                        <IconText icon={statusObj.icon} text={statusObj.statusText} />
                    </Label>
                    <Label>Cluster: {clusterName}</Label>
                </LabelGroup>
            )}
            <ToggleGroup aria-label="Toggle for check details modal view" className="pf-v5-u-py-sm">
                <ToggleGroupItem
                    text="Check details"
                    buttonId="check-details-toggle-group"
                    isSelected
                />
                <ToggleGroupItem
                    text="Remediation details"
                    buttonId="remediation-details-toggle-group"
                />
            </ToggleGroup>
            <Divider component="div" className="pf-v5-u-pb-md" />
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
                actions={[
                    <Button key="cancel" variant="primary" onClick={handleClose}>
                        Close
                    </Button>,
                ]}
            >
                <DescriptionList>
                    <DescriptionListGroup>
                        <DescriptionListTerm>Description</DescriptionListTerm>
                        <DescriptionListDescription className="formatted-text">
                            {description}
                        </DescriptionListDescription>
                    </DescriptionListGroup>
                    <DescriptionListGroup>
                        <DescriptionListTerm>Rationale</DescriptionListTerm>
                        <DescriptionListDescription className="formatted-text">
                            {rationale}
                        </DescriptionListDescription>
                    </DescriptionListGroup>
                    <DescriptionListGroup>
                        <DescriptionListTerm>Instructions</DescriptionListTerm>
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
