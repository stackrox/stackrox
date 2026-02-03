import { Alert } from '@patternfly/react-core';

function VirtualMachineScanScopeAlert() {
    return (
        <Alert
            isInline
            component="p"
            variant="info"
            title="The results show only DNF packages from Red Hat repositories. System packages preinstalled with the VM image are scanned only after registration (for example, 'subscription-manager register' or 'rhc connect') and at least one DNF transaction (for example, 'dnf install' or 'dnf update'). Scan results refresh every 4 hours by default."
        />
    );
}

export default VirtualMachineScanScopeAlert;
