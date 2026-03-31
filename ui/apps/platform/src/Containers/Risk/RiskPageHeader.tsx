import { Flex, PageSection, Title } from '@patternfly/react-core';

import useIsRouteEnabled from 'hooks/useIsRouteEnabled';
import usePermissions from 'hooks/usePermissions';

import CreatePolicyFromSearch from './CreatePolicyFromSearch';

function RiskPageHeader() {
    const isRouteEnabled = useIsRouteEnabled();
    const { hasReadWriteAccess } = usePermissions();
    // Require READ_WRITE_ACCESS to create plus READ_ACCESS to other resources for Policies route.
    const hasWriteAccessForCreatePolicy =
        hasReadWriteAccess('WorkflowAdministration') && isRouteEnabled('policy-management');

    return (
        <PageSection>
            <Flex
                alignItems={{ default: 'alignItemsCenter' }}
                justifyContent={{ default: 'justifyContentSpaceBetween' }}
            >
                <Title headingLevel="h1">Risk</Title>
                {hasWriteAccessForCreatePolicy && <CreatePolicyFromSearch />}
            </Flex>
        </PageSection>
    );
}

export default RiskPageHeader;
