import React from 'react';
import {
    AngleDoubleDownIcon,
    AngleDoubleUpIcon,
    CriticalRiskIcon,
    EqualsIcon,
    UnknownIcon,
} from '@patternfly/react-icons';
import { SVGIconProps } from '@patternfly/react-icons/dist/esm/createIcon';

import { VulnerabilitySeverity } from 'types/cve.proto';
import { vulnSeverityIconColors } from 'constants/visuals/colors';

// Defines the default PF icons that represent a CVE severity, and sets the default colors for the icons.
// Color can be overridden by passing the standard `color` prop to the icon component.
const SeverityIcons: Record<VulnerabilitySeverity, React.FC<SVGIconProps>> = {
    CRITICAL_VULNERABILITY_SEVERITY: (props) => (
        <CriticalRiskIcon
            {...props}
            color={props.color ?? vulnSeverityIconColors.CRITICAL_VULNERABILITY_SEVERITY}
        />
    ),
    IMPORTANT_VULNERABILITY_SEVERITY: (props) => (
        <AngleDoubleUpIcon
            {...props}
            color={props.color ?? vulnSeverityIconColors.IMPORTANT_VULNERABILITY_SEVERITY}
        />
    ),
    MODERATE_VULNERABILITY_SEVERITY: (props) => (
        <EqualsIcon
            {...props}
            color={props.color ?? vulnSeverityIconColors.MODERATE_VULNERABILITY_SEVERITY}
        />
    ),
    LOW_VULNERABILITY_SEVERITY: (props) => (
        <AngleDoubleDownIcon
            {...props}
            color={props.color ?? vulnSeverityIconColors.LOW_VULNERABILITY_SEVERITY}
        />
    ),
    UNKNOWN_VULNERABILITY_SEVERITY: (props) => (
        <UnknownIcon
            {...props}
            color={props.color ?? vulnSeverityIconColors.UNKNOWN_VULNERABILITY_SEVERITY}
        />
    ),
};

export default SeverityIcons;
