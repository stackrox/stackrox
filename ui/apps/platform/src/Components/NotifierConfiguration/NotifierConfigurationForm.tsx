import React, { useEffect, useState } from 'react';
import type { ReactElement } from 'react';
import { Button, Card, CardBody, CardTitle, Flex, FlexItem, Tooltip } from '@patternfly/react-core';
import { HelpIcon, PencilAltIcon, PlusCircleIcon, TrashIcon } from '@patternfly/react-icons';
import type { FormikErrors } from 'formik';

import { isDefaultEmailTemplate } from 'Components/EmailTemplate/EmailTemplate.utils';
import EmailTemplateModal from 'Components/EmailTemplate/EmailTemplateModal';
import type { TemplatePreviewArgs } from 'Components/EmailTemplate/EmailTemplateModal';
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
    customBodyDefault: string;
    customSubjectDefault: string;
    errors: FormikErrors<unknown>;
    // Caller provides name of property in formik.values and PatternFly fieldId props.
    // For example:
    // 'deliveryDestinations' for Vulnerability Reports
    // 'report.notifierConfigurations' for Compliance Reports
    fieldIdPrefixForFormikAndPatternFly: string;
    hasWriteAccessForIntegration: boolean;
    notifierConfigurations: NotifierConfiguration[];
    onDeleteLastNotifierConfiguration?: () => void;
    renderTemplatePreview?: (args: TemplatePreviewArgs) => ReactElement;
    setFieldValue: (fieldId: string, value: unknown) => void;
};

function NotifierConfigurationForm({
    customBodyDefault,
    customSubjectDefault,
    errors,
    fieldIdPrefixForFormikAndPatternFly,
    hasWriteAccessForIntegration,
    notifierConfigurations,
    onDeleteLastNotifierConfiguration,
    renderTemplatePreview,
    setFieldValue,
}: NotifierConfigurationFormProps): ReactElement {
    const { keyFor } = useIndexKey();
    const [notifiers, setNotifiers] = useState<NotifierIntegrationBase[]>([]);
    const [isLoadingNotifiers, setIsLoadingNotifiers] = useState(false);
    const [notifierConfigurationSelected, setNotifierConfigurationSelected] =
        useState<NotifierConfiguration | null>(null);

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
            <ul>
                {notifierConfigurations.map((notifierConfiguration, index) => {
                    const { emailConfig, notifierName } = notifierConfiguration;
                    const { customBody, customSubject, mailingLists, notifierId } = emailConfig;
                    const fieldId = `${fieldIdPrefixForFormikAndPatternFly}[${index}]`;
                    const isDefaultEmailTemplateApplied = isDefaultEmailTemplate({
                        customBody,
                        customSubject,
                    });
                    return (
                        <li key={keyFor(index)} className="pf-v5-u-mb-md">
                            <Card>
                                <CardTitle>
                                    <Flex
                                        alignItems={{
                                            default: 'alignItemsCenter',
                                        }}
                                    >
                                        <FlexItem flex={{ default: 'flex_1' }}>
                                            Delivery destination
                                        </FlexItem>
                                        <FlexItem>
                                            <Button
                                                variant="plain"
                                                aria-label="Delete delivery destination"
                                                onClick={() => {
                                                    const notifierConfigurationsFiltered =
                                                        notifierConfigurations.filter(
                                                            (notifierConfigurationArg) =>
                                                                notifierConfigurationArg !==
                                                                notifierConfiguration
                                                        );
                                                    setFieldValue(
                                                        fieldIdPrefixForFormikAndPatternFly,
                                                        notifierConfigurationsFiltered
                                                    );
                                                    setMailingListsStrings(
                                                        mailingListsStrings.filter(
                                                            (_, i) => i !== index
                                                        )
                                                    );
                                                    if (
                                                        notifierConfigurationsFiltered.length ===
                                                            0 &&
                                                        onDeleteLastNotifierConfiguration
                                                    ) {
                                                        onDeleteLastNotifierConfiguration();
                                                    }
                                                }}
                                            >
                                                <TrashIcon />
                                            </Button>
                                        </FlexItem>
                                    </Flex>
                                </CardTitle>
                                <CardBody>
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
                                    <div className="pf-v5-u-mt-md">
                                        <FormLabelGroup
                                            label="Email template"
                                            labelIcon={
                                                <Tooltip
                                                    content={
                                                        isDefaultEmailTemplateApplied ? (
                                                            <div>
                                                                Default template applied. Edit to
                                                                customize.
                                                            </div>
                                                        ) : (
                                                            <div>
                                                                Custom template applied. Edit to
                                                                customize.
                                                            </div>
                                                        )
                                                    }
                                                >
                                                    <Button
                                                        variant="plain"
                                                        aria-label="More info for email template field"
                                                        aria-describedby={`${fieldId}.customSubject`}
                                                    >
                                                        <HelpIcon aria-label="More info for email template field" />
                                                    </Button>
                                                </Tooltip>
                                            }
                                            fieldId={`${fieldId}.customSubject`}
                                            errors={errors}
                                            isRequired
                                        >
                                            <Button
                                                variant="link"
                                                isInline
                                                icon={<PencilAltIcon />}
                                                onClick={() => {
                                                    setNotifierConfigurationSelected(
                                                        notifierConfiguration
                                                    );
                                                }}
                                                iconPosition="right"
                                            >
                                                {isDefaultEmailTemplateApplied
                                                    ? 'Default template applied'
                                                    : 'Custom template applied'}
                                            </Button>
                                        </FormLabelGroup>
                                    </div>
                                </CardBody>
                            </Card>
                        </li>
                    );
                })}
                <li>
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
                        Add delivery destination
                    </Button>
                </li>
            </ul>
            {notifierConfigurationSelected && (
                <EmailTemplateModal
                    customBodyDefault={customBodyDefault}
                    customBodyInitial={notifierConfigurationSelected.emailConfig.customBody}
                    customSubjectDefault={customSubjectDefault}
                    customSubjectInitial={notifierConfigurationSelected.emailConfig.customSubject}
                    onChange={({ customBody, customSubject }) => {
                        const index = notifierConfigurations.indexOf(notifierConfigurationSelected);
                        if (index >= 0) {
                            const { emailConfig } = notifierConfigurationSelected;
                            setFieldValue(`${fieldIdPrefixForFormikAndPatternFly}[${index}]`, {
                                ...notifierConfigurationSelected,
                                emailConfig: {
                                    ...emailConfig,
                                    customSubject,
                                    customBody,
                                },
                            });
                        }
                    }}
                    onClose={() => {
                        setNotifierConfigurationSelected(null);
                    }}
                    renderTemplatePreview={renderTemplatePreview}
                    title="Edit email template"
                />
            )}
        </>
    );
}

export default NotifierConfigurationForm;
