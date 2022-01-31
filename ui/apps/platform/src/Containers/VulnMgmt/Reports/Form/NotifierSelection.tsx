import React, { useState, useEffect, ReactElement } from 'react';
import { Link } from 'react-router-dom';
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
import { integrationsPath } from 'routePaths';
import { NotifierIntegration } from 'types/notifier.proto';

type NotifierSelectionProps = {
    notifierId: string;
    mailingLists: string[];
    setFieldValue: (field: string, value: any, shouldValidate?: boolean | undefined) => void;
    handleBlur: (e: React.FocusEvent<any, Element>) => void;
    errors: FormikErrors<any>;
    touched: FormikTouched<any>;
};

function NotifierSelection({
    notifierId,
    mailingLists,
    setFieldValue,
    handleBlur,
    errors,
    touched,
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
                <FlexItem>
                    <Button
                        className="pf-u-mb-md"
                        variant={ButtonVariant.secondary}
                        component={(props) => (
                            <Link {...props} to={`${integrationsPath}/notifiers/email/create`} />
                        )}
                    >
                        Create email notifier
                    </Button>
                </FlexItem>
            </Flex>
            <FormLabelGroup
                label="Distribution list"
                fieldId="emailConfig.mailingLists"
                touched={touched}
                errors={errors}
                helperText="Enter an audience, who will receive the scheduled report. If an audience is not entered, the recipient defined in the notifier will be used. Multiple email addresses can be entered with comma separators."
            >
                <TextInput
                    type="text"
                    id="emailConfig.mailingLists"
                    value={joinedMailingLists}
                    onChange={onMailingListsChange}
                    onBlur={handleBlur}
                    placeholder="annie@example.com,jack@example.com"
                />
            </FormLabelGroup>
        </>
    );
}

export default NotifierSelection;
