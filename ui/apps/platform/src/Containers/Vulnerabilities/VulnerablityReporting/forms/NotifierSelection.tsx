import React, { useState, useEffect, ReactElement } from 'react';
import { Button, Flex, FlexItem, TextInput } from '@patternfly/react-core';
import { SelectOption } from '@patternfly/react-core/deprecated';
import { FormikProps } from 'formik';
import isEqual from 'lodash/isEqual';
import resolvePath from 'object-resolve-path';

import EmailNotifierModal from 'Components/EmailNotifier/EmailNotifierModal';
import SelectSingle from 'Components/SelectSingle';
import FormLabelGroup from 'Components/PatternFly/FormLabelGroup';
import { fetchIntegration } from 'services/IntegrationsService';
import { NotifierIntegration } from 'types/notifier.proto';
import { ReportFormValues } from './useReportFormValues';

type ReportNotifier = {
    id: string;
    name: string;
};

type NotifierSelectionProps = {
    prefixId: string;
    selectedNotifier: ReportNotifier | null;
    mailingLists: string[];
    allowCreate: boolean;
    formik: FormikProps<ReportFormValues>;
};

function NotifierSelection({
    prefixId,
    selectedNotifier,
    mailingLists,
    allowCreate,
    formik,
}: NotifierSelectionProps): ReactElement {
    const [notifiers, setNotifiers] = useState<NotifierIntegration[]>([]);
    const [lastAddedNotifier, setLastAddedNotifier] = useState<NotifierIntegration | null>(null);
    const [isEmailNotifierModalOpen, setIsEmailNotifierModalOpen] = useState(false);

    useEffect(() => {
        fetchIntegration('notifiers')
            .then((response) => {
                const notifiersList =
                    (response?.response?.notifiers as NotifierIntegration[]) || [];
                const emailNotifiers = notifiersList.filter(
                    (notifier) => notifier.type === 'email'
                );
                setNotifiers(emailNotifiers);

                if (lastAddedNotifier) {
                    onNotifierChange('notifier', lastAddedNotifier);
                }
            })
            .catch(() => {
                // TODO display message when there is a place for minor errors
            });
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [lastAddedNotifier]);

    function onToggleEmailNotifierModal() {
        setIsEmailNotifierModalOpen((current) => !current);
    }

    function onMailingListsChange(value) {
        const explodedEmails: string = value.split(',').map((email) => email.trim() as string);
        formik.setFieldValue(`${prefixId}.emailConfig.mailingLists`, explodedEmails);
    }

    function onNotifierChange(_id, selectionId) {
        const notifierObject = notifiers.find((notifier) => notifier.id === selectionId);
        if (notifierObject) {
            const notifierMailingLists = notifierObject.labelDefault.split(',');
            const deliveryDestinationPrev = resolvePath(formik.values, prefixId);
            const { emailConfig: emailConfigPrev } = deliveryDestinationPrev;
            formik.setFieldValue(prefixId, {
                emailConfig: {
                    ...emailConfigPrev,
                    notifierId: notifierObject.id,
                    mailingLists: mailingLists.length === 0 ? notifierMailingLists : mailingLists,
                },
                notifierName: notifierObject.name,
            });
            setIsEmailNotifierModalOpen(false);
        }
    }

    function getDefaultNotifierMailingLists(): string[] | null {
        if (selectedNotifier) {
            const notifierObject = notifiers.find(
                (notifier) => notifier.id === selectedNotifier.id
            );
            if (notifierObject) {
                return notifierObject.labelDefault.split(',');
            }
        }
        return null;
    }

    function onSetToDefaultNotifierMailingLists() {
        const notifierMailingLists = getDefaultNotifierMailingLists();
        if (notifierMailingLists) {
            formik.setFieldValue(`${prefixId}.emailConfig.mailingLists`, notifierMailingLists);
        }
    }

    const joinedMailingLists = mailingLists.join(', ');
    const notifierMailingLists = getDefaultNotifierMailingLists();

    const isResetToDefaultDisabled = isEqual(mailingLists, notifierMailingLists);

    return (
        <>
            <FormLabelGroup
                className="pf-v5-u-mb-md"
                isRequired
                label="Email notifier"
                fieldId={`${prefixId}.notifier`}
                errors={formik.errors}
            >
                <Flex direction={{ default: 'row' }} alignItems={{ default: 'alignItemsFlexEnd' }}>
                    <FlexItem>
                        <SelectSingle
                            id={`${prefixId}.notifier`}
                            toggleAriaLabel="Select a notifier"
                            value={selectedNotifier?.id || ''}
                            handleSelect={onNotifierChange}
                            placeholderText="Select a notifier"
                            footer={
                                allowCreate && (
                                    <Button
                                        variant="link"
                                        isInline
                                        onClick={onToggleEmailNotifierModal}
                                    >
                                        Create email notifier
                                    </Button>
                                )
                            }
                        >
                            {notifiers.map(({ id, name }) => (
                                <SelectOption key={id} value={id}>
                                    {name}
                                </SelectOption>
                            ))}
                        </SelectSingle>
                    </FlexItem>
                </Flex>
            </FormLabelGroup>
            <FormLabelGroup
                isRequired
                label="Distribution list"
                fieldId={`${prefixId}.emailConfig.mailingLists`}
                helperText="Enter an audience, who will receive the scheduled report. Multiple email addresses can be entered with comma separators."
                errors={formik.errors}
            >
                <TextInput
                    isRequired
                    type="text"
                    value={joinedMailingLists}
                    onChange={(_event, value) => onMailingListsChange(value)}
                    placeholder="annie@example.com,jack@example.com"
                />
            </FormLabelGroup>
            {selectedNotifier && (
                <Button
                    className="pf-v5-u-mt-sm"
                    variant="link"
                    isInline
                    size="sm"
                    onClick={onSetToDefaultNotifierMailingLists}
                    isDisabled={isResetToDefaultDisabled}
                >
                    Reset to default
                </Button>
            )}
            <EmailNotifierModal
                isOpen={isEmailNotifierModalOpen}
                updateNotifierList={setLastAddedNotifier}
                onToggleEmailNotifierModal={onToggleEmailNotifierModal}
            />
        </>
    );
}

export default NotifierSelection;
