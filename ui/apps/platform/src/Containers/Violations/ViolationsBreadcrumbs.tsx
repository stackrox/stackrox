import type { ReactElement } from 'react';
import { Breadcrumb, BreadcrumbItem } from '@patternfly/react-core';

import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import type { FilteredWorkflowView } from 'Components/FilteredWorkflowViewSelector/types';

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
        <Breadcrumb>
            {topLevelBreadcrumb}
            {subPageBreadcrumb}
        </Breadcrumb>
    );
};

export default ViolationsBreadcrumbs;
