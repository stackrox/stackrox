import React from 'react';
import {
    PageSection,
    Title,
    Divider,
    Flex,
    FlexItem,
    Breadcrumb,
    BreadcrumbItem,
    Wizard,
} from '@patternfly/react-core';

import { vulnerabilityReportsPath } from 'routePaths';

import PageTitle from 'Components/PageTitle';
import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import ReportParametersForm from '../forms/ReportParametersForm';

function VulnReportsPage() {
    return (
        <>
            <PageTitle title="Create vulnerability report" />
            <PageSection variant="light" className="pf-u-py-md">
                <Breadcrumb>
                    <BreadcrumbItemLink to={vulnerabilityReportsPath}>
                        Vulnerability reporting
                    </BreadcrumbItemLink>
                    <BreadcrumbItem isActive>Create report</BreadcrumbItem>
                </Breadcrumb>
            </PageSection>
            <Divider component="div" />
            <PageSection variant="light" padding={{ default: 'noPadding' }}>
                <Flex direction={{ default: 'column' }} className="pf-u-py-lg pf-u-px-lg">
                    <FlexItem>
                        <Title headingLevel="h1">Create report</Title>
                    </FlexItem>
                    <FlexItem>
                        Configure reports, define report scopes, and assign distribution lists to
                        report on vulnerabilities across the organization.
                    </FlexItem>
                </Flex>
            </PageSection>
            <Divider component="div" />
            <PageSection padding={{ default: 'noPadding' }} isCenterAligned>
                <Wizard
                    // @TODO: Make the navAriaLabel dynamic based on whether you're creating, editing, or cloning
                    navAriaLabel="Report creation steps"
                    // @TODO: Make the mainAriaLabel dynamic based on whether you're creating, editing, or cloning
                    mainAriaLabel="Report creation content"
                    steps={[
                        {
                            name: 'Configure report parameters',
                            component: <ReportParametersForm />,
                        },
                        { name: 'Configure delivery destinations (Optional)', component: <p /> },
                        {
                            name: 'Review and create',
                            component: <p />,
                            nextButtonText: 'Finish',
                        },
                    ]}
                    hasNoBodyPadding
                />
            </PageSection>
        </>
    );
}

export default VulnReportsPage;
