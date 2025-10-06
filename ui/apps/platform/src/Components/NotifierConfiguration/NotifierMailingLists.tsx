import React, { useState, ReactElement } from 'react';
import { Button, Flex, FlexItem, TextInput, SelectOption } from '@patternfly/react-core';
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
    // Caller provides name of property in formik.values and PatternFly fieldId props.
    // For example:
    // 'deliveryDestinations[0]' for Vulnerability Reports
    // 'report.notifierConfigurations[0]' for Compliance Reports
    fieldIdPrefixForFormikAndPatternFly: string;
    hasWriteAccessForIntegration: boolean;
    isLoadingNotifiers: boolean;
    mailingListsString: string;
    notifierId: string;
    notifierName: string;
    notifiers: NotifierIntegrationBase[];
    setMailingLists: (mailingListsString: string) => void;
    setNotifier: (notifier: NotifierIntegrationBase) => void;
    setNotifiers: (notifiers: NotifierIntegrationBase[]) => void;
};

function NotifierMailingLists({
    errors,
    fieldIdPrefixForFormikAndPatternFly,
    hasWriteAccessForIntegration,
    isLoadingNotifiers,
    mailingListsString,
    notifierId,
    notifierName,
    notifiers,
    setMailingLists,
    setNotifier,
    setNotifiers,
}: NotifierMailingListsProps): ReactElement {
    const [isEmailNotifierModalOpen, setIsEmailNotifierModalOpen] = useState(false);

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

    // Special case to prevent initial temporary render of id instead of name.
    const selectOptions =
        isLoadingNotifiers && notifierId
            ? [
                  <SelectOption key={notifierId} value={notifierId}>
                      {notifierName}
                  </SelectOption>,
              ]
            : notifiers.map(({ id, name }) => (
                  <SelectOption key={id} value={id}>
                      {name}
                  </SelectOption>
              ));

    return (
        <>
            <FormLabelGroup
                className="pf-v5-u-mb-md"
                isRequired
                label="Email notifier"
                fieldId={`${fieldIdPrefixForFormikAndPatternFly}.notifier`}
                errors={errors}
            >
                <Flex direction={{ default: 'row' }} alignItems={{ default: 'alignItemsFlexEnd' }}>
                    <FlexItem>
                        <SelectSingle
                            id={`${fieldIdPrefixForFormikAndPatternFly}.notifier`}
                            isDisabled={isLoadingNotifiers}
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
                            {selectOptions}
                        </SelectSingle>
                    </FlexItem>
                </Flex>
            </FormLabelGroup>
            <FormLabelGroup
                isRequired
                label="Distribution list"
                fieldId={`${fieldIdPrefixForFormikAndPatternFly}.emailConfig.mailingLists`}
                helperText="Enter an audience, who will receive the scheduled report. Multiple email addresses can be entered with comma separators."
                errors={errors}
            >
                <TextInput
                    isRequired
                    type="text"
                    id={`${fieldIdPrefixForFormikAndPatternFly}.emailConfig.mailingLists`}
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
