import { useState } from 'react';
import type { ReactElement } from 'react';
import { Button, Flex, FlexItem, Title } from '@patternfly/react-core';
import { Table, Tbody, Thead, Td, Th, Tr } from '@patternfly/react-table';

import { isDefaultEmailTemplate } from 'Components/EmailTemplate/EmailTemplate.utils';
import EmailTemplateModal from 'Components/EmailTemplate/EmailTemplateModal';
import type { TemplatePreviewArgs } from 'Components/EmailTemplate/EmailTemplateModal';
import useIndexKey from 'hooks/useIndexKey';
import type { NotifierConfiguration } from 'services/ReportsService.types';

export type NotifierConfigurationViewProps = {
    headingLevel: 'h2' | 'h3';
    customBodyDefault: string;
    customSubjectDefault: string;
    notifierConfigurations: NotifierConfiguration[];
    renderTemplatePreview?: (args: TemplatePreviewArgs) => ReactElement;
};

function NotifierConfigurationView({
    headingLevel,
    customBodyDefault,
    customSubjectDefault,
    notifierConfigurations,
    renderTemplatePreview,
}: NotifierConfigurationViewProps): ReactElement {
    const { keyFor } = useIndexKey();
    const [notifierConfigurationSelected, setNotifierConfigurationSelected] =
        useState<NotifierConfiguration | null>(null);

    return (
        <>
            <Flex direction={{ default: 'column' }}>
                <FlexItem>
                    <Title headingLevel={headingLevel}>Delivery destinations</Title>
                </FlexItem>
                <FlexItem flex={{ default: 'flexNone' }}>
                    {notifierConfigurations.length === 0 ? (
                        'No delivery destinations'
                    ) : (
                        <Table variant="compact">
                            <Thead>
                                <Tr>
                                    <Th>Email notifier</Th>
                                    <Th>Distribution list</Th>
                                    <Th>Email template</Th>
                                </Tr>
                            </Thead>
                            <Tbody>
                                {notifierConfigurations.map((notifierConfiguration, index) => {
                                    const { emailConfig, notifierName } = notifierConfiguration;
                                    const { customBody, customSubject, mailingLists } = emailConfig;
                                    const isDefaultEmailTemplateApplied = isDefaultEmailTemplate({
                                        customBody,
                                        customSubject,
                                    });
                                    return (
                                        <Tr key={keyFor(index)}>
                                            <Td dataLabel="Email notifier">{notifierName}</Td>
                                            <Td dataLabel="Distribution list">
                                                {mailingLists.join(', ')}
                                            </Td>
                                            <Td dataLabel="Email template">
                                                <Button
                                                    variant="link"
                                                    isInline
                                                    onClick={() => {
                                                        setNotifierConfigurationSelected(
                                                            notifierConfiguration
                                                        );
                                                    }}
                                                >
                                                    {isDefaultEmailTemplateApplied
                                                        ? 'Default template applied'
                                                        : 'Custom template applied'}
                                                </Button>
                                            </Td>
                                        </Tr>
                                    );
                                })}
                            </Tbody>
                        </Table>
                    )}
                </FlexItem>
            </Flex>
            {notifierConfigurationSelected && (
                <EmailTemplateModal
                    customBodyDefault={customBodyDefault}
                    customBodyInitial={notifierConfigurationSelected.emailConfig.customBody}
                    customSubjectDefault={customSubjectDefault}
                    customSubjectInitial={notifierConfigurationSelected.emailConfig.customSubject}
                    onChange={null}
                    onClose={() => {
                        setNotifierConfigurationSelected(null);
                    }}
                    renderTemplatePreview={renderTemplatePreview}
                    title={
                        isDefaultEmailTemplate({
                            customBody: notifierConfigurationSelected.emailConfig.customBody,
                            customSubject: notifierConfigurationSelected.emailConfig.customSubject,
                        })
                            ? 'Default template applied'
                            : 'Custom template applied'
                    }
                />
            )}
        </>
    );
}

export default NotifierConfigurationView;
