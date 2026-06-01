import { useLocation } from 'react-router-dom-v5-compat';
import { Nav, NavItem, NavList } from '@patternfly/react-core';
import { Link } from 'react-router-dom-v5-compat';

import {
    vulnerabilitiesPrototypeCvePath,
    vulnerabilitiesPrototypeAdvisoriesPath,
    vulnerabilitiesPrototypeComponentsPath,
    vulnerabilitiesPrototypeDeploymentsPath,
} from 'routePaths';

/**
 * Horizontal navigation bar for the CVE prototype pages.
 * Renders tabs for CVEs, Advisories, and Deployments.
 */
function ProtoNav() {
    const location = useLocation();
    const isCves =
        location.pathname.includes('/cves') ||
        location.pathname.endsWith('/prototype');
    const isAdvisories = location.pathname.includes('/advisories');
    const isComponents = location.pathname.includes('/components');
    const isDeployments = location.pathname.includes('/deployments');

    return (
        <Nav variant="horizontal">
            <NavList>
                <NavItem isActive={isCves}>
                    <Link to={vulnerabilitiesPrototypeCvePath}>CVEs</Link>
                </NavItem>
                <NavItem isActive={isAdvisories}>
                    <Link to={vulnerabilitiesPrototypeAdvisoriesPath}>
                        Advisories
                    </Link>
                </NavItem>
                <NavItem isActive={isComponents}>
                    <Link to={vulnerabilitiesPrototypeComponentsPath}>
                        Components
                    </Link>
                </NavItem>
                <NavItem isActive={isDeployments}>
                    <Link to={vulnerabilitiesPrototypeDeploymentsPath}>
                        Deployments
                    </Link>
                </NavItem>
            </NavList>
        </Nav>
    );
}

export default ProtoNav;
