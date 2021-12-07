import React, { useState, useEffect, ReactElement } from 'react';
import { SelectOption, Text, TextInput, TextVariants, Title } from '@patternfly/react-core';

import SelectSingle from 'Components/SelectSingle';
import FormLabelGroup from 'Components/PatternFly/FormLabelGroup';
import { fetchIntegration } from 'services/IntegrationsService';
import { NotifierIntegration } from 'types/notifier.proto';

type NotifierSelectionProps = {
    notifierId: string;
    mailingLists: string[];
    setFieldValue: (field: string, value: any, shouldValidate?: boolean | undefined) => void;
    handleBlur: (e: React.FocusEvent<any, Element>) => void;
};

function NotifierSelection({
    notifierId,
    mailingLists,
    setFieldValue,
    handleBlur,
}: NotifierSelectionProps): ReactElement {
    const [notifiers, setNotifiers] = useState<NotifierIntegration[]>([]);

    function fetchNotifiers(): void {
        fetchIntegration('notifiers')
            .then((response) => {
                const notifiersList =
                    (response?.response?.notifiers as NotifierIntegration[]) || [];
                const emailNotifiers = notifiersList.filter(
                    (notifier) => notifier.type === 'email'
                );
                setNotifiers(emailNotifiers);
            })
            .catch(() => {
                // TODO display message when there is a place for minor errors
            });
    }

    useEffect(() => {
        fetchNotifiers();
    }, []);

    function onMailingListsChange(value) {
        const explodedEmails = value.split(',').map((email) => email.trim() as string);
        setFieldValue('notifierConfig.emailConfig.mailingLists', explodedEmails);
    }

    function onNotifierChange(_id, selection) {
        setFieldValue('notifierConfig.emailConfig.notifierId', selection);
    }

    const joinedMailingLists = mailingLists.join(', ');

    return (
        <>
            <Title headingLevel="h2" className="pf-u-mb-xs">
                Notification method and distribution
            </Title>
            <Text component={TextVariants.p} className="pf-u-mb-md">
                Schedule reports across the organization by defining a notification method and
                distribution list for the report
            </Text>
            <FormLabelGroup
                className="pf-u-mb-md"
                isRequired
                label="Notifier"
                fieldId="notifierConfig.emailConfig.notifierId"
                touched={{}}
                errors={{}}
            >
                <SelectSingle
                    id="notifierConfig.emailConfig.notifierId"
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
            <FormLabelGroup
                label="Distribution list"
                fieldId="notifierConfig.emailConfig.mailingLists"
                touched={{}}
                errors={{}}
                helperText="Enter an audience, who will receive the scheduled report. If an audience is not entered, the recipient defined in the notifier will be used. Multiple addresses can be entered with comma separators."
            >
                <TextInput
                    type="text"
                    id="notifierConfig.emailConfig.mailingLists"
                    value={joinedMailingLists}
                    onChange={onMailingListsChange}
                    onBlur={handleBlur}
                />
            </FormLabelGroup>
        </>
    );
}

export default NotifierSelection;
