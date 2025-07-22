import React, { ReactElement } from 'react';

import { ReportFormValues } from 'Containers/Vulnerabilities/VulnerablityReporting/forms/useReportFormValues';
import { Divider, Flex, FlexItem, PageSection, Title } from '@patternfly/react-core';

import { TemplatePreviewArgs } from 'Components/EmailTemplate/EmailTemplateModal';
import NotifierConfigurationView from 'Components/NotifierConfiguration/NotifierConfigurationView';

import EmailTemplatePreview from '../components/EmailTemplatePreview';
import ReportParametersDetails from '../components/ReportParametersDetails';
import ScheduleDetails from '../components/ScheduleDetails';
import { defaultEmailBody, getDefaultEmailSubject } from './emailTemplateFormUtils';

export type ReportReviewFormProps = {
    title: string;
    formValues: ReportFormValues;
};

const headingLevel = 'h3';

function ReportReviewForm({ title, formValues }: ReportReviewFormProps): ReactElement {
    return (
        <>
            <PageSection variant="light" padding={{ default: 'noPadding' }}>
                <Flex direction={{ default: 'column' }} className="pf-v5-u-py-lg pf-v5-u-px-lg">
                    <FlexItem>
                        <Title headingLevel="h2">{title}</Title>
                    </FlexItem>
                </Flex>
            </PageSection>
            <Divider component="div" />
            <PageSection
                variant="light"
                padding={{ default: 'noPadding' }}
                className="pf-v5-u-py-lg pf-v5-u-px-lg"
            >
                <ReportParametersDetails headingLevel={headingLevel} formValues={formValues} />
                <Divider component="div" className="pf-v5-u-py-md" />
                <NotifierConfigurationView
                    headingLevel={headingLevel}
                    customBodyDefault={defaultEmailBody}
                    customSubjectDefault={getDefaultEmailSubject(
                        formValues.reportParameters.reportName,
                        formValues.reportParameters.reportScope?.name
                    )}
                    notifierConfigurations={formValues.deliveryDestinations}
                    renderTemplatePreview={({
                        customBody,
                        customSubject,
                        customSubjectDefault,
                    }: TemplatePreviewArgs) => (
                        <EmailTemplatePreview
                            emailSubject={customSubject}
                            emailBody={customBody}
                            defaultEmailSubject={customSubjectDefault}
                            reportParameters={formValues.reportParameters}
                        />
                    )}
                />
                <Divider component="div" className="pf-v5-u-py-md" />
                <ScheduleDetails formValues={formValues} />
            </PageSection>
        </>
    );
}

export default ReportReviewForm;
