import { useParams } from 'react-router-dom-v5-compat';
import { Spinner } from '@patternfly/react-core';
import { ExclamationCircleIcon } from '@patternfly/react-icons';

import EmptyStateTemplate from 'Components/EmptyStateTemplate';
import useURLPagination from 'hooks/useURLPagination';
import useURLSearch from 'hooks/useURLSearch';
import { DEFAULT_VM_PAGE_SIZE } from 'Containers/Vulnerabilities/constants';
import { WorkloadCveViewContext } from 'Containers/Vulnerabilities/WorkloadCves/WorkloadCveViewContext';
import DeploymentPageVulnerabilities from 'Containers/Vulnerabilities/WorkloadCves/Deployment/DeploymentPageVulnerabilities';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import { useDefaultWorkloadCveViewContext } from '../hooks/useDefaultWorkloadCveViewContext';
import { useWorkloadId } from '../hooks/useWorkloadId';
import { useAnalyticsPageView } from '../hooks/useAnalyticsPageView';

export function WorkloadSecurityTab() {
    useAnalyticsPageView();

    const context = useDefaultWorkloadCveViewContext();
    const pagination = useURLPagination(DEFAULT_VM_PAGE_SIZE);
    const { searchFilter, setSearchFilter } = useURLSearch();
    const { ns, name } = useParams();
    const { id, isLoading, error } = useWorkloadId({ ns, name });

    return (
        <WorkloadCveViewContext.Provider value={context}>
            {isLoading && <Spinner aria-label="Loading workload security data" />}
            {error && (
                <EmptyStateTemplate
                    headingLevel={'h2'}
                    title={`Unable to find security data for workload "${name}" in namespace "${ns}"`}
                    icon={ExclamationCircleIcon}
                    iconClassName="pf-v6-u-danger-color-100"
                >
                    {error.message ?? getAxiosErrorMessage(error)}
                </EmptyStateTemplate>
            )}
            {id && (
                <DeploymentPageVulnerabilities
                    deploymentId={id}
                    pagination={pagination}
                    showVulnerabilityStateTabs={false}
                    vulnerabilityState="OBSERVED"
                    searchFilter={searchFilter}
                    setSearchFilter={setSearchFilter}
                />
            )}
        </WorkloadCveViewContext.Provider>
    );
}
