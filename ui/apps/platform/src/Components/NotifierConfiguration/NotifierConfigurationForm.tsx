import { useEffect, useState } from 'react';
import type { ReactElement } from 'react';
import { Button, Flex, FormSection, TextArea, TextInput } from '@patternfly/react-core';
import { PlusCircleIcon, TrashIcon } from '@patternfly/react-icons';
import type { FormikErrors } from 'formik';

import FormLabelGroup from 'Components/PatternFly/FormLabelGroup';
import useIndexKey from 'hooks/useIndexKey';
import { fetchNotifierIntegrations } from 'services/NotifierIntegrationsService';
import type { NotifierIntegrationBase } from 'services/NotifierIntegrationsService';
import type { NotifierConfiguration } from 'services/ReportsService.types';

import NotifierMailingLists from './NotifierMailingLists';

function isEmailNotifier(notifier: NotifierIntegrationBase) {
    return notifier.type === 'email' || notifier.type === 'acscsEmail';
}

function splitAndTrimMailingListsString(mailingListsString: string): string[] {
    return mailingListsString
        .split(',')
        .map((email) => email.trim())
        .filter((email) => email.length !== 0);
}

export type NotifierConfigurationFormProps = {
    errors: FormikErrors<unknown>;
    // Caller provides name of property in formik.values and PatternFly fieldId props.
    // For example:
    // 'deliveryDestinations' for Vulnerability Reports
    // 'report.notifierConfigurations' for Compliance Reports
    fieldIdPrefixForFormikAndPatternFly: string;
    hasWriteAccessForIntegration: boolean;
    notifierConfigurations: NotifierConfiguration[];
    onDeleteLastNotifierConfiguration?: () => void;
    setFieldValue: (fieldId: string, value: unknown) => void;
};

function NotifierConfigurationForm({
    errors,
    fieldIdPrefixForFormikAndPatternFly,
    hasWriteAccessForIntegration,
    notifierConfigurations,
    onDeleteLastNotifierConfiguration,
    setFieldValue,
}: NotifierConfigurationFormProps): ReactElement {
    const { keyFor } = useIndexKey();
    const [notifiers, setNotifiers] = useState<NotifierIntegrationBase[]>([]);
    const [isLoadingNotifiers, setIsLoadingNotifiers] = useState(false);

    useEffect(() => {
        setIsLoadingNotifiers(true);
        fetchNotifierIntegrations()
            .then((notifiersFetched) => {
                setNotifiers(notifiersFetched.filter(isEmailNotifier));
            })
            .catch(() => {
                // TODO display message when there is a place for minor errors
            })
            .finally(() => {
                setIsLoadingNotifiers(false);
            });
    }, []);

    // Initialize rare second source of truth for TextInput elements.
    // Array length corresponds to number of delivery destinations.
    const [mailingListsStrings, setMailingListsStrings] = useState<string[]>(
        notifierConfigurations.map(({ emailConfig }) => emailConfig.mailingLists.join(', '))
    );

    // Update string value for TextInput element independently of mailingLists string array in formik state.
    //
    // The original split-trim-join round trip for mailingLists as single source of truth had two problems:
    //
    // On one side of the coin, join appends comma and space, which immediately replaced a deleted final space,
    // therefore prevented user from backspacing to delete a comma after the last non-empty string.
    //
    // On other side of the coin, filter (not in original) to omit empty string items (which fail backend validation)
    // would immediately omit a final comma.
    function updateMailingListsString(index: number, mailingListsStringUpdated: string) {
        setMailingListsStrings(
            mailingListsStrings.map((mailingListsString, i) =>
                i === index ? mailingListsStringUpdated : mailingListsString
            )
        );
    }

    return (
        <>
            {notifierConfigurations.map((notifierConfiguration, index) => {
                const { emailConfig, notifierName } = notifierConfiguration;
                const { customBody, customSubject, mailingLists, notifierId } = emailConfig;
                const fieldId = `${fieldIdPrefixForFormikAndPatternFly}[${index}]`;

                return (
                    <FormSection key={keyFor(index)} title="Destination" titleElement="h3">
                        <Flex direction={{ default: 'row' }}>
                            <Button
                                variant="link"
                                icon={<TrashIcon />}
                                onClick={() => {
                                    const notifierConfigurationsFiltered =
                                        notifierConfigurations.filter(
                                            (notifierConfigurationArg) =>
                                                notifierConfigurationArg !== notifierConfiguration
                                        );
                                    setFieldValue(
                                        fieldIdPrefixForFormikAndPatternFly,
                                        notifierConfigurationsFiltered
                                    );
                                    setMailingListsStrings(
                                        mailingListsStrings.filter((_, i) => i !== index)
                                    );
                                    if (
                                        notifierConfigurationsFiltered.length === 0 &&
                                        onDeleteLastNotifierConfiguration
                                    ) {
                                        onDeleteLastNotifierConfiguration();
                                    }
                                }}
                            >
                                Delete destination
                            </Button>
                        </Flex>
                        <NotifierMailingLists
                            errors={errors}
                            fieldIdPrefixForFormikAndPatternFly={fieldId}
                            hasWriteAccessForIntegration={hasWriteAccessForIntegration}
                            isLoadingNotifiers={isLoadingNotifiers}
                            mailingListsString={mailingListsStrings[index]}
                            notifierId={notifierId}
                            notifierName={notifierName}
                            notifiers={notifiers}
                            setMailingLists={(mailingListsString: string) => {
                                setFieldValue(
                                    `${fieldId}.emailConfig.mailingLists`,
                                    splitAndTrimMailingListsString(mailingListsString)
                                );
                                updateMailingListsString(index, mailingListsString);
                            }}
                            setNotifier={(notifier: NotifierIntegrationBase) => {
                                setFieldValue(fieldId, {
                                    emailConfig: {
                                        ...emailConfig,
                                        notifierId: notifier.id,
                                        mailingLists:
                                            mailingLists.length === 0
                                                ? splitAndTrimMailingListsString(
                                                      notifier.labelDefault
                                                  )
                                                : mailingLists,
                                    },
                                    notifierName: notifier.name,
                                });
                                updateMailingListsString(index, notifier.labelDefault);
                            }}
                            setNotifiers={setNotifiers}
                        />
                        <FormLabelGroup
                            label="Custom subject"
                            fieldId="customSubject"
                            errors={errors}
                        >
                            <TextInput
                                type="text"
                                id="customSubject"
                                name="customSubject"
                                value={customSubject}
                                onChange={(_event, value) => {
                                    setFieldValue(`${fieldId}.customSubject`, value);
                                }}
                            />
                        </FormLabelGroup>
                        <FormLabelGroup label="Custom body" fieldId="customBody" errors={errors}>
                            <TextArea
                                type="text"
                                id="customBody"
                                name="customBody"
                                value={customBody}
                                onChange={(_event, value) => {
                                    setFieldValue(`${fieldId}.customBody`, value);
                                }}
                            />
                        </FormLabelGroup>
                    </FormSection>
                );
            })}
            <Flex direction={{ default: 'row' }}>
                <Button
                    variant="link"
                    icon={<PlusCircleIcon />}
                    onClick={() => {
                        const notifierConfiguration: NotifierConfiguration = {
                            emailConfig: {
                                notifierId: '',
                                mailingLists: [],
                                customSubject: '',
                                customBody: '',
                            },
                            notifierName: '',
                        };
                        setFieldValue(fieldIdPrefixForFormikAndPatternFly, [
                            ...notifierConfigurations,
                            notifierConfiguration,
                        ]);
                        setMailingListsStrings([...mailingListsStrings, '']);
                    }}
                >
                    Add destination
                </Button>
            </Flex>
        </>
    );
}

export default NotifierConfigurationForm;
