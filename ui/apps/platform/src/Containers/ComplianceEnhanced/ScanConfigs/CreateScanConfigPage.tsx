import React, { ReactElement } from 'react';

import { Breadcrumb, BreadcrumbItem, Divider, PageSection, Title } from '@patternfly/react-core';

import { complianceEnhancedScanConfigsBasePath } from 'routePaths';
import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import PageTitle from 'Components/PageTitle';

import ScanConfigWizardForm from './Wizard/ScanConfigWizardForm';

function ScanConfigPage(): ReactElement {
    return (
        <>
            <PageTitle title="Compliance Scan Configuration" />
            <PageSection variant="light" className="pf-u-py-md">
                <Breadcrumb>
                    <BreadcrumbItemLink to={complianceEnhancedScanConfigsBasePath}>
                        Scan schedules
                    </BreadcrumbItemLink>
                    <BreadcrumbItem isActive>Create scan schedule</BreadcrumbItem>
                </Breadcrumb>
            </PageSection>
            <Divider component="div" />
            <PageSection variant="light" padding={{ default: 'noPadding' }}>
                <Title headingLevel="h1" className="pf-u-py-lg pf-u-px-lg">
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

export default ScanConfigPage;
