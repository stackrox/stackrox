import React, { useState, useEffect, ReactElement } from 'react';

import {
    Button,
    ButtonVariant,
    Flex,
    FlexItem,
    FormGroup,
    SelectOption,
    TextInput,
} from '@patternfly/react-core';

import SelectSingle from 'Components/SelectSingle';
import FormLabelGroup from 'Components/PatternFly/FormLabelGroup';
import { fetchIntegration } from 'services/IntegrationsService';
import { NotifierIntegration } from 'types/notifier.proto';
// eslint-disable-next-line import/no-named-as-default
import EmailNotifierFormModal from './EmailNotifierFormModal';

type NotifierSelectionProps = {
    selectedNotifier: NotifierIntegration | null;
    mailingLists: string[];
    setFieldValue: (field: string, value: any, shouldValidate?: boolean | undefined) => void;
    allowCreate: boolean;
};

function NotifierSelection({
    selectedNotifier,
    mailingLists,
    setFieldValue,
    allowCreate,
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
        const explodedEmails = value.split(',').map((email) => email.trim() as string);
        setFieldValue('mailingLists', explodedEmails);
    }

    function onNotifierChange(_id, selectionId) {
        const notifierObject = notifiers.find((notifier) => notifier.id === selectionId);
        if (notifierObject) {
            setFieldValue('notifier', notifierObject);
            setIsEmailNotifierModalOpen(false);
        }
    }

    const joinedMailingLists = mailingLists.join(', ');

    return (
        <>
            <Flex alignItems={{ default: 'alignItemsFlexEnd' }}>
                <FlexItem>
                    <FormLabelGroup
                        className="pf-u-mb-md"
                        isRequired
                        label="Email notifier"
                        fieldId="notifierId"
                        touched={{}}
                        errors={{}}
                    >
                        <SelectSingle
                            id="notifierId"
                            toggleAriaLabel="Select a notifier"
                            value={selectedNotifier?.id || ''}
                            handleSelect={onNotifierChange}
                            placeholderText="Select a notifier"
                            isDisabled={notifiers.length === 0}
                        >
                            {notifiers.map(({ id, name }) => (
                                <SelectOption key={id} value={id}>
                                    {name}
                                </SelectOption>
                            ))}
                        </SelectSingle>
                    </FormLabelGroup>
                </FlexItem>
                {allowCreate && (
                    <FlexItem>
                        <Button
                            className="pf-u-mb-md"
                            variant={ButtonVariant.secondary}
                            onClick={onToggleEmailNotifierModal}
                        >
                            Create email notifier
                        </Button>
                    </FlexItem>
                )}
            </Flex>
            <FormGroup
                isRequired
                label="Distribution list"
                fieldId="mailingLists"
                helperText="Enter an audience, who will receive the scheduled report. Multiple email addresses can be entered with comma separators."
            >
                <TextInput
                    isRequired
                    type="text"
                    id="mailingLists"
                    value={joinedMailingLists}
                    onChange={onMailingListsChange}
                    placeholder="annie@example.com,jack@example.com"
                />
            </FormGroup>
            <EmailNotifierFormModal
                isOpen={isEmailNotifierModalOpen}
                updateNotifierList={setLastAddedNotifier}
                onToggleEmailNotifierModal={onToggleEmailNotifierModal}
            />
        </>
    );
}

export default NotifierSelection;
