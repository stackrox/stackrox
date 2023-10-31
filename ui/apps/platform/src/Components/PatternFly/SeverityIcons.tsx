import React from 'react';
import {
    AngleDoubleDownIcon,
    AngleDoubleUpIcon,
    CriticalRiskIcon,
    EqualsIcon,
    UnknownIcon,
} from '@patternfly/react-icons';
import { SVGIconProps } from '@patternfly/react-icons/dist/esm/createIcon';

import {
    CRITICAL_SEVERITY_COLOR,
    LOW_SEVERITY_COLOR,
    IMPORTANT_HIGH_SEVERITY_COLOR,
    MODERATE_MEDIUM_SEVERITY_COLOR,
    UNKNOWN_SEVERITY_COLOR,
} from 'constants/severityColors';
import { VulnerabilitySeverity } from 'types/cve.proto';
import { PolicySeverity } from 'types/policy.proto';

// Defines the default PF icons that represent a CVE severity, and sets the default colors for the icons.
// Color can be overridden by passing the standard `color` prop to the icon component.
// For example, colors={count === 0 ? noViolationsColor: undefined} prop.

export const CriticalSeverityIcon = (props) => (
    <CriticalRiskIcon {...props} color={props.color ?? CRITICAL_SEVERITY_COLOR} />
);

export const ImportantSeverityIcon = (props) => (
    <AngleDoubleUpIcon {...props} color={props.color ?? IMPORTANT_HIGH_SEVERITY_COLOR} />
);

export const HighSeverityIcon = ImportantSeverityIcon; // High is for policy severity

export const ModerateSeverityIcon = (props) => (
    <EqualsIcon {...props} color={props.color ?? MODERATE_MEDIUM_SEVERITY_COLOR} />
);

export const MediumSeverityIcon = ModerateSeverityIcon; // Medium is for policy severity

export const LowSeverityIcon = (props) => (
    <AngleDoubleDownIcon {...props} color={props.color ?? LOW_SEVERITY_COLOR} />
);

export const UnknownSeverityIcon = (props) => (
    <UnknownIcon {...props} color={props.color ?? UNKNOWN_SEVERITY_COLOR} />
);

const SeverityIcons: Record<VulnerabilitySeverity, React.FC<React.PropsWithChildren<SVGIconProps>>> = {
    CRITICAL_VULNERABILITY_SEVERITY: CriticalSeverityIcon,
    IMPORTANT_VULNERABILITY_SEVERITY: ImportantSeverityIcon,
    MODERATE_VULNERABILITY_SEVERITY: ModerateSeverityIcon,
    LOW_VULNERABILITY_SEVERITY: LowSeverityIcon,
    UNKNOWN_VULNERABILITY_SEVERITY: UnknownSeverityIcon,
};

export const policySeverityIconMap: Record<PolicySeverity, React.FC<React.PropsWithChildren<SVGIconProps>>> = {
    CRITICAL_SEVERITY: CriticalSeverityIcon,
    HIGH_SEVERITY: HighSeverityIcon,
    MEDIUM_SEVERITY: MediumSeverityIcon,
    LOW_SEVERITY: LowSeverityIcon,
};

export default SeverityIcons;
