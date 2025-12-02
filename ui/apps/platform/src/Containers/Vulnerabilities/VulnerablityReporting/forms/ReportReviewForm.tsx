import type { ReactElement } from 'react';

import { Divider, Flex, FlexItem, PageSection, Title } from '@patternfly/react-core';

import type { TemplatePreviewArgs } from 'Components/EmailTemplate/EmailTemplateModal';
import NotifierConfigurationView from 'Components/NotifierConfiguration/NotifierConfigurationView';

import EmailTemplatePreview from '../components/EmailTemplatePreview';
import ReportParametersDetails from '../components/ReportParametersDetails';
import ScheduleDetails from '../components/ScheduleDetails';
import { defaultEmailBody, getDefaultEmailSubject } from './emailTemplateFormUtils';
import type { ReportFormValues } from './useReportFormValues';

export type ReportReviewFormProps = {
    title: string;
    formValues: ReportFormValues;
};

const headingLevel = 'h3';

function ReportReviewForm({ title, formValues }: ReportReviewFormProps): ReactElement {
    return (
        <>
            <PageSection hasBodyWrapper={false} padding={{ default: 'noPadding' }}>
                <Flex direction={{ default: 'column' }} className="pf-v6-u-py-lg pf-v6-u-px-lg">
                    <FlexItem>
                        <Title headingLevel="h2">{title}</Title>
                    </FlexItem>
                </Flex>
            </PageSection>
            <Divider component="div" />
            <PageSection
                hasBodyWrapper={false}
                padding={{ default: 'noPadding' }}
                className="pf-v6-u-py-lg pf-v6-u-px-lg"
            >
                <ReportParametersDetails headingLevel={headingLevel} formValues={formValues} />
                <Divider component="div" className="pf-v6-u-py-md" />
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
                <Divider component="div" className="pf-v6-u-py-md" />
                <ScheduleDetails formValues={formValues} />
            </PageSection>
        </>
    );
}

export default ReportReviewForm;
