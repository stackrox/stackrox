import React from 'react';
import { Breadcrumb, BreadcrumbItem, Divider, PageSection } from '@patternfly/react-core';
import { useParams } from 'react-router-dom';

import PageTitle from 'Components/PageTitle';
import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import { complianceEnhancedCoveragePath } from 'routePaths';

function CheckDetails() {
    const { checkName, profileName } = useParams();

    const complianceCoverageChecksURL = `${complianceEnhancedCoveragePath}/profiles/${profileName}/checks`;

    return (
        <>
            <PageTitle title="Compliance coverage - Check" />
            <PageSection variant="light" className="pf-v5-u-py-md">
                <Breadcrumb>
                    <BreadcrumbItem>Compliance coverage</BreadcrumbItem>
                    <BreadcrumbItemLink to={complianceCoverageChecksURL}>
                        {profileName}
                    </BreadcrumbItemLink>
                    <BreadcrumbItem isActive>{checkName}</BreadcrumbItem>
                </Breadcrumb>
            </PageSection>
            <Divider component="div" />
        </>
    );
}

export default CheckDetails;
