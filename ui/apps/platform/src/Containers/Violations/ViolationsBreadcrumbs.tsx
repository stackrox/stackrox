import { Breadcrumb, BreadcrumbItem, Divider, PageSection } from '@patternfly/react-core';
import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import React, { ReactElement } from 'react';

import { violationsBasePath } from 'routePaths';

type ViolationsBreadcrumbsProps = {
    /** The title of the current Violation entity sub-page */
    current?: string;
};

const ViolationsBreadcrumbs = ({ current }: ViolationsBreadcrumbsProps): ReactElement => {
    const topLevelBreadcrumb = current ? (
        <BreadcrumbItemLink to={violationsBasePath}>Violations</BreadcrumbItemLink>
    ) : (
        <BreadcrumbItem>Violations</BreadcrumbItem>
    );
    const subPageBreadcrumb = current ? <BreadcrumbItem isActive>{current}</BreadcrumbItem> : '';

    return (
        <>
            <PageSection variant="light" className="pf-v5-u-py-md">
                <Breadcrumb className="pf-v5-u-mb-0 pf-v5-u-pl-0">
                    {topLevelBreadcrumb}
                    {subPageBreadcrumb}
                </Breadcrumb>
            </PageSection>
            <Divider component="div" />
        </>
    );
};

export default ViolationsBreadcrumbs;
