import {
    Button,
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    Flex,
    FlexItem,
    Modal,
    Title,
} from '@patternfly/react-core';
import React, { ReactElement } from 'react';

import { ReportFormValues } from 'Containers/Vulnerabilities/VulnerablityReporting/forms/useReportFormValues';
import useIndexKey from 'hooks/useIndexKey';
import { getDefaultEmailSubject, isDefaultEmailTemplate } from '../forms/emailTemplateFormUtils';
import useEmailTemplateModal from '../hooks/useEmailTemplateModal';
import EmailTemplatePreview from './EmailTemplatePreview';

export type DeliveryDestinationsDetailsProps = {
    formValues: ReportFormValues;
};

function DeliveryDestinationsDetails({
    formValues,
}: DeliveryDestinationsDetailsProps): ReactElement {
    const { keyFor } = useIndexKey();
    const {
        isEmailTemplateModalOpen,
        closeEmailTemplateModal,
        selectedEmailSubject,
        selectedEmailBody,
        setSelectedDeliveryDestination,
    } = useEmailTemplateModal();

    const deliveryDestinations =
        formValues.deliveryDestinations.length !== 0 ? (
            formValues.deliveryDestinations.map((deliveryDestination) => (
                <li key={deliveryDestination.notifier?.id}>{deliveryDestination.notifier?.name}</li>
            ))
        ) : (
            <li>None</li>
        );

    const mailingLists =
        formValues.deliveryDestinations.length !== 0 ? (
            formValues.deliveryDestinations.map((deliveryDestination) => {
                const emails = deliveryDestination?.mailingLists.join(', ');
                return <li key={emails}>{emails}</li>;
            })
        ) : (
            <li>None</li>
        );

    const emailTemplates =
        formValues.deliveryDestinations.length !== 0 ? (
            formValues.deliveryDestinations.map((deliveryDestination, index) => {
                const { customSubject, customBody } = deliveryDestination;
                const isDefaultEmailTemplateApplied = isDefaultEmailTemplate(
                    customSubject,
                    customBody
                );
                return (
                    <li key={keyFor(index)}>
                        <Button
                            variant="link"
                            isInline
                            onClick={() => {
                                setSelectedDeliveryDestination(deliveryDestination);
                            }}
                            iconPosition="right"
                        >
                            {isDefaultEmailTemplateApplied
                                ? 'Default template applied'
                                : 'Custom template applied'}
                        </Button>
                    </li>
                );
            })
        ) : (
            <li>None</li>
        );

    const defaultSelectedEmailSubject = getDefaultEmailSubject(
        formValues.reportParameters.reportName,
        formValues.reportParameters.reportScope?.name
    );

    const modalTitle = isDefaultEmailTemplate(selectedEmailSubject, selectedEmailBody)
        ? 'Default template applied'
        : 'Custom template applied';

    return (
        <>
            <Flex direction={{ default: 'column' }}>
                <FlexItem>
                    <Title headingLevel="h3">Delivery destinations</Title>
                </FlexItem>
                <FlexItem flex={{ default: 'flexNone' }}>
                    <DescriptionList
                        columnModifier={{
                            default: '3Col',
                        }}
                    >
                        <DescriptionListGroup>
                            <DescriptionListTerm>Email notifier</DescriptionListTerm>
                            <DescriptionListDescription>
                                <ul>{deliveryDestinations}</ul>
                            </DescriptionListDescription>
                        </DescriptionListGroup>
                        <DescriptionListGroup>
                            <DescriptionListTerm>Distribution list</DescriptionListTerm>
                            <DescriptionListDescription>
                                <ul>{mailingLists}</ul>
                            </DescriptionListDescription>
                        </DescriptionListGroup>
                        <DescriptionListGroup>
                            <DescriptionListTerm>Email template</DescriptionListTerm>
                            <DescriptionListDescription>
                                <ul>{emailTemplates}</ul>
                            </DescriptionListDescription>
                        </DescriptionListGroup>
                    </DescriptionList>
                </FlexItem>
            </Flex>
            <Modal
                variant="medium"
                title={modalTitle}
                isOpen={isEmailTemplateModalOpen}
                onClose={closeEmailTemplateModal}
                actions={[
                    <Button key="cancel" variant="primary" onClick={closeEmailTemplateModal}>
                        Close
                    </Button>,
                ]}
            >
                <EmailTemplatePreview
                    emailSubject={selectedEmailSubject}
                    emailBody={selectedEmailBody}
                    defaultEmailSubject={defaultSelectedEmailSubject}
                    reportParameters={formValues.reportParameters}
                />
            </Modal>
        </>
    );
}

export default DeliveryDestinationsDetails;
