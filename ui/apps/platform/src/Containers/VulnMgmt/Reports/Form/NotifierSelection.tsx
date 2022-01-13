import React, { useState, useEffect, ReactElement } from 'react';
import { SelectOption, TextInput } from '@patternfly/react-core';

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
        setFieldValue('emailConfig.mailingLists', explodedEmails);
    }

    function onNotifierChange(_id, selection) {
        setFieldValue('emailConfig.notifierId', selection);
    }

    const joinedMailingLists = mailingLists.join(', ');

    return (
        <>
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
                fieldId="emailConfig.mailingLists"
                touched={{}}
                errors={{}}
                helperText="Enter an audience, who will receive the scheduled report. If an audience is not entered, the recipient defined in the notifier will be used. Multiple addresses can be entered with comma separators."
            >
                <TextInput
                    type="text"
                    id="emailConfig.mailingLists"
                    value={joinedMailingLists}
                    onChange={onMailingListsChange}
                    onBlur={handleBlur}
                />
            </FormLabelGroup>
        </>
    );
}

export default NotifierSelection;
