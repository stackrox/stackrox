import React, { useState, useEffect, ReactElement } from 'react';
import { FormikErrors, FormikTouched } from 'formik';

import {
    Button,
    ButtonVariant,
    Flex,
    FlexItem,
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
    notifierId: string;
    mailingLists: string[];
    setFieldValue: (field: string, value: unknown, shouldValidate?: boolean | undefined) => void;
    handleBlur: (e: React.FocusEvent<unknown, Element>) => void;
    errors: FormikErrors<unknown>;
    touched: FormikTouched<unknown>;
    allowCreate: boolean;
};

function NotifierSelection({
    notifierId,
    mailingLists,
    setFieldValue,
    handleBlur,
    errors,
    touched,
    allowCreate,
}: NotifierSelectionProps): ReactElement {
    const [notifiers, setNotifiers] = useState<NotifierIntegration[]>([]);
    const [lastAddedNotifierId, setLastAddedNotifierId] = useState('');
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

                if (lastAddedNotifierId) {
                    onNotifierChange('emailConfig.notifierId', lastAddedNotifierId);
                }
            })
            .catch(() => {
                // TODO display message when there is a place for minor errors
            });
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [lastAddedNotifierId]);

    function onToggleEmailNotifierModal() {
        setIsEmailNotifierModalOpen((current) => !current);
    }

    function onMailingListsChange(value) {
        const explodedEmails = value.split(',').map((email) => email.trim() as string);
        setFieldValue('emailConfig.mailingLists', explodedEmails);
    }

    function onNotifierChange(_id, selection) {
        setFieldValue('emailConfig.notifierId', selection);
    }

    const joinedMailingLists = mailingLists.join(', ');

    return (
        <>
            <Flex alignItems={{ default: 'alignItemsFlexEnd' }}>
                <FlexItem>
                    <FormLabelGroup
                        className="pf-u-mb-md"
                        isRequired
                        label="Notifier"
                        fieldId="emailConfig.notifierId"
                        touched={{}}
                        errors={{}}
                    >
                        <SelectSingle
                            id="emailConfig.notifierId"
                            toggleAriaLabel="Select a notifier"
                            value={notifierId}
                            handleSelect={onNotifierChange}
                            placeholderText="Select a notifier"
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
            <FormLabelGroup
                isRequired
                label="Distribution list"
                fieldId="emailConfig.mailingLists"
                touched={touched}
                errors={errors}
                helperText="Enter an audience, who will receive the scheduled report. Multiple email addresses can be entered with comma separators."
            >
                <TextInput
                    isRequired
                    type="text"
                    id="emailConfig.mailingLists"
                    value={joinedMailingLists}
                    onChange={onMailingListsChange}
                    onBlur={handleBlur}
                    placeholder="annie@example.com,jack@example.com"
                />
            </FormLabelGroup>
            <EmailNotifierFormModal
                isOpen={isEmailNotifierModalOpen}
                updateNotifierList={setLastAddedNotifierId}
                onToggleEmailNotifierModal={onToggleEmailNotifierModal}
            />
        </>
    );
}

export default NotifierSelection;
