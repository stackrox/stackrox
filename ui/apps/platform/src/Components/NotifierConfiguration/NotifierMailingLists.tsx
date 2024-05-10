import React, { useState, useEffect, ReactElement } from 'react';
import { Button, Flex, FlexItem, TextInput } from '@patternfly/react-core';
import { SelectOption } from '@patternfly/react-core/deprecated';
import { FormikErrors } from 'formik';

import EmailNotifierModal from 'Components/EmailNotifier/EmailNotifierModal';
import SelectSingle from 'Components/SelectSingle';
import FormLabelGroup from 'Components/PatternFly/FormLabelGroup';
import {
    NotifierIntegrationBase,
    fetchNotifierIntegrations,
} from 'services/NotifierIntegrationsService';

function isEmailNotifier(notifier: NotifierIntegrationBase) {
    return notifier.type === 'email';
}

type NotifierMailingListsProps = {
    errors: FormikErrors<unknown>;
    fieldIdPrefix: string;
    hasWriteAccessForIntegration: boolean;
    mailingLists: string[];
    notifierId: string;
    setMailingLists: (mailingListsString: string) => void;
    setNotifier: (notifier: NotifierIntegrationBase) => void;
};

function NotifierMailingLists({
    errors,
    fieldIdPrefix,
    hasWriteAccessForIntegration,
    mailingLists,
    notifierId,
    setMailingLists,
    setNotifier,
}: NotifierMailingListsProps): ReactElement {
    const [notifiers, setNotifiers] = useState<NotifierIntegrationBase[]>([]);
    const [isEmailNotifierModalOpen, setIsEmailNotifierModalOpen] = useState(false);

    useEffect(() => {
        fetchNotifierIntegrations()
            .then((notifiersFetched) => {
                setNotifiers(notifiersFetched.filter(isEmailNotifier));
            })
            .catch(() => {
                // TODO display message when there is a place for minor errors
            });
    }, []);

    function updateNotifierList(notifierAdded: NotifierIntegrationBase) {
        fetchNotifierIntegrations()
            .then((notifiersFetched) => {
                setNotifiers(notifiersFetched.filter(isEmailNotifier));
                setNotifier(notifierAdded);
                setIsEmailNotifierModalOpen(false);
            })
            .catch(() => {
                // TODO display message when there is a place for minor errors
            });
    }

    function onToggleEmailNotifierModal() {
        setIsEmailNotifierModalOpen((current) => !current);
    }

    function onSelectNotifier(_id, selectionId) {
        const notifierSelected = notifiers.find((notifier) => notifier.id === selectionId);
        if (notifierSelected) {
            setNotifier(notifierSelected);
        }
    }

    function getMailingListsStringFromNotifier(): string {
        if (notifierId) {
            const notifierSelected = notifiers.find((notifier) => notifier.id === notifierId);
            if (notifierSelected) {
                return notifierSelected.labelDefault;
            }
        }
        return '';
    }

    function onSetToDefaultNotifierMailingLists() {
        const mailingListsStringFromNotifier = getMailingListsStringFromNotifier();
        if (mailingListsStringFromNotifier) {
            setMailingLists(mailingListsStringFromNotifier);
        }
    }

    const mailingListsString = mailingLists.join(', ');

    return (
        <>
            <FormLabelGroup
                className="pf-v5-u-mb-md"
                isRequired
                label="Email notifier"
                fieldId={`${fieldIdPrefix}.notifier`}
                errors={errors}
            >
                <Flex direction={{ default: 'row' }} alignItems={{ default: 'alignItemsFlexEnd' }}>
                    <FlexItem>
                        <SelectSingle
                            id={`${fieldIdPrefix}.notifier`}
                            toggleAriaLabel="Select a notifier"
                            value={notifierId}
                            handleSelect={onSelectNotifier}
                            placeholderText="Select a notifier"
                            footer={
                                hasWriteAccessForIntegration && (
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
                fieldId={`${fieldIdPrefix}.emailConfig.mailingLists`}
                helperText="Enter an audience, who will receive the scheduled report. Multiple email addresses can be entered with comma separators."
                errors={errors}
            >
                <TextInput
                    isRequired
                    type="text"
                    value={mailingListsString}
                    onChange={(_event, value) => setMailingLists(value)}
                    placeholder="annie@example.com,jack@example.com"
                />
            </FormLabelGroup>
            {!!notifierId && (
                <Button
                    className="pf-v5-u-mt-sm"
                    variant="link"
                    isInline
                    size="sm"
                    onClick={onSetToDefaultNotifierMailingLists}
                    isDisabled={mailingListsString === getMailingListsStringFromNotifier()}
                >
                    Reset to default
                </Button>
            )}
            <EmailNotifierModal
                isOpen={isEmailNotifierModalOpen}
                updateNotifierList={updateNotifierList}
                onToggleEmailNotifierModal={onToggleEmailNotifierModal}
            />
        </>
    );
}

export default NotifierMailingLists;
