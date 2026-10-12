import type { ReactElement } from 'react';

import { Breadcrumb, BreadcrumbItem, PageSection, Title } from '@patternfly/react-core';

import { complianceEnhancedSchedulesPath } from 'routePaths';
import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import PageTitle from 'Components/PageTitle';

import ScanConfigWizardPageSection from './Wizard/ScanConfigWizardPageSection';

function CreateScanConfigPage(): ReactElement {
    return (
        <>
            <PageTitle title="Create Compliance Scan Configuration" />
            <PageSection type="breadcrumb">
                <Breadcrumb>
                    <BreadcrumbItemLink to={complianceEnhancedSchedulesPath}>
                        Scan schedules
                    </BreadcrumbItemLink>
                    <BreadcrumbItem isActive>Create scan schedule</BreadcrumbItem>
                </Breadcrumb>
            </PageSection>
            <PageSection>
                <Title headingLevel="h1">Create scan schedule</Title>
            </PageSection>
            <ScanConfigWizardPageSection pageAction="create" />
        </>
    );
}

export default CreateScanConfigPage;
