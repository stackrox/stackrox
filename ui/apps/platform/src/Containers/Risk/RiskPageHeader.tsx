import PageHeader from 'Components/PageHeader';
import useIsRouteEnabled from 'hooks/useIsRouteEnabled';
import usePermissions from 'hooks/usePermissions';

import CreatePolicyFromSearch from './CreatePolicyFromSearch';

type RiskPageHeaderProps = {
    isViewFiltered: boolean;
};

function RiskPageHeader({ isViewFiltered }: RiskPageHeaderProps) {
    const isRouteEnabled = useIsRouteEnabled();
    const { hasReadWriteAccess } = usePermissions();
    // Require READ_WRITE_ACCESS to create plus READ_ACCESS to other resources for Policies route.
    const hasWriteAccessForCreatePolicy =
        hasReadWriteAccess('WorkflowAdministration') && isRouteEnabled('policy-management');

    const subHeader = isViewFiltered ? 'Filtered view' : 'Default view';
    return (
        <PageHeader header="Risk" subHeader={subHeader}>
            {hasWriteAccessForCreatePolicy && <CreatePolicyFromSearch />}
        </PageHeader>
    );
}

export default RiskPageHeader;
