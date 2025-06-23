import React, { ReactElement } from 'react';

import { Breadcrumb, BreadcrumbItem, Divider, PageSection, Title } from '@patternfly/react-core';

import { complianceEnhancedSchedulesPath } from 'routePaths';
import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import PageTitle from 'Components/PageTitle';

import ScanConfigWizardForm from './Wizard/ScanConfigWizardForm';

function CreateScanConfigPage(): ReactElement {
    return (
        <>
            <PageTitle title="Compliance Scan Configuration" />
            <PageSection variant="light" className="pf-v5-u-py-md">
                <Breadcrumb>
                    <BreadcrumbItemLink to={complianceEnhancedSchedulesPath}>
                        Scan schedules
                    </BreadcrumbItemLink>
                    <BreadcrumbItem isActive>Create scan schedule</BreadcrumbItem>
                </Breadcrumb>
            </PageSection>
            <Divider component="div" />
            <PageSection variant="light" padding={{ default: 'noPadding' }}>
                <Title headingLevel="h1" className="pf-v5-u-py-lg pf-v5-u-px-lg">
                    Create scan schedule
                </Title>
            </PageSection>
            <Divider component="div" />
            <PageSection padding={{ default: 'noPadding' }} isCenterAligned>
                <ScanConfigWizardForm />
            </PageSection>
        </>
    );
}

export default CreateScanConfigPage;
