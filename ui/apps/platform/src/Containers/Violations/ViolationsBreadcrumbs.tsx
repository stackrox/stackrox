import React, { ReactElement } from 'react';
import { Breadcrumb, BreadcrumbItem, Divider, PageSection } from '@patternfly/react-core';

import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import { FilteredWorkflowView } from 'Components/FilteredWorkflowViewSelector/types';

import {
    violationsFullViewPath,
    violationsPlatformViewPath,
    violationsUserWorkloadsViewPath,
} from 'routePaths';
import { ensureExhaustive } from 'utils/type.utils';

function getTopLevelBreadcrumb(filteredWorkflowView: FilteredWorkflowView) {
    switch (filteredWorkflowView) {
        case 'Applications view':
            return {
                title: 'User workload violations',
                url: violationsUserWorkloadsViewPath,
            };
        case 'Platform view':
            return {
                title: 'Platform violations',
                url: violationsPlatformViewPath,
            };
        case 'Full view':
            return {
                title: 'All violations',
                url: violationsFullViewPath,
            };
        default:
            return ensureExhaustive(filteredWorkflowView);
    }
}

type ViolationsBreadcrumbsProps = {
    /** The title of the current Violation entity sub-page */
    current?: string;
    /** The current Violation sub-page workflow filter */
    filteredWorkflowView: FilteredWorkflowView;
};

const ViolationsBreadcrumbs = ({
    current,
    filteredWorkflowView,
}: ViolationsBreadcrumbsProps): ReactElement => {
    const { title, url } = getTopLevelBreadcrumb(filteredWorkflowView);
    const topLevelBreadcrumb = current ? (
        <BreadcrumbItemLink to={url}>{title}</BreadcrumbItemLink>
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
