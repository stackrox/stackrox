import React, { useState, useEffect, ReactElement } from 'react';

import {
    Button,
    ButtonVariant,
    Flex,
    FlexItem,
    SelectOption,
    TextInput,
} from '@patternfly/react-core';
import { FormikProps } from 'formik';

import SelectSingle from 'Components/SelectSingle';
import FormLabelGroup from 'Components/PatternFly/FormLabelGroup';
import { fetchIntegration } from 'services/IntegrationsService';
import { NotifierIntegration } from 'types/notifier.proto';
import { ReportFormValues, ReportNotifier } from './useReportFormValues';

// eslint-disable-next-line import/no-named-as-default
import EmailNotifierFormModal from './EmailNotifierFormModal';

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
        formik.setFieldValue(`${prefixId}.mailingLists`, explodedEmails);
    }

    function onNotifierChange(_id, selectionId) {
        const notifierObject = notifiers.find((notifier) => notifier.id === selectionId);
        if (notifierObject) {
            const notifierMailingLists = notifierObject.labelDefault.split(',');
            formik.setFieldValue(prefixId, {
                notifier: notifierObject,
                mailingLists: notifierMailingLists,
            });
            setIsEmailNotifierModalOpen(false);
        }
    }

    const joinedMailingLists = mailingLists.join(', ');
    return (
        <>
            <FormLabelGroup
                className="pf-u-mb-md"
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
                            isDisabled={notifiers.length === 0}
                        >
                            {notifiers.map(({ id, name }) => (
                                <SelectOption key={id} value={id}>
                                    {name}
                                </SelectOption>
                            ))}
                        </SelectSingle>
                    </FlexItem>
                    {allowCreate && (
                        <FlexItem>
                            <Button
                                variant={ButtonVariant.secondary}
                                onClick={onToggleEmailNotifierModal}
                            >
                                Create email notifier
                            </Button>
                        </FlexItem>
                    )}
                </Flex>
            </FormLabelGroup>
            <FormLabelGroup
                isRequired
                label="Distribution list"
                fieldId={`${prefixId}.mailingLists`}
                helperText="Enter an audience, who will receive the scheduled report. Multiple email addresses can be entered with comma separators."
                errors={formik.errors}
            >
                <TextInput
                    isRequired
                    type="text"
                    value={joinedMailingLists}
                    onChange={onMailingListsChange}
                    placeholder="annie@example.com,jack@example.com"
                />
            </FormLabelGroup>
            <EmailNotifierFormModal
                isOpen={isEmailNotifierModalOpen}
                updateNotifierList={setLastAddedNotifier}
                onToggleEmailNotifierModal={onToggleEmailNotifierModal}
            />
        </>
    );
}

export default NotifierSelection;
