import React from 'react';
import { Alert } from '@patternfly/react-core';

function VirtualMachineScanScopeAlert() {
    return (
        <Alert
            isInline
            component="p"
            variant="info"
            title="The results only show vulnerabilities for DNF packages, that come from Red Hat repositories. The scan doesn't include System packages , which are preinstalled with the VM image and aren't tracked by the DNF database. Any DNF update could impact this default behavior."
        />
    );
}

export default VirtualMachineScanScopeAlert;
