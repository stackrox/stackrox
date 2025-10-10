import React from 'react';
import type { FC, PropsWithChildren } from 'react';
import {
    AngleDoubleDownIcon,
    AngleDoubleUpIcon,
    CriticalRiskIcon,
    EqualsIcon,
    UnknownIcon,
} from '@patternfly/react-icons';
import type { SVGIconProps } from '@patternfly/react-icons/dist/esm/createIcon';
import { Icon } from '@patternfly/react-core';

import {
    CRITICAL_SEVERITY_COLOR,
    LOW_SEVERITY_COLOR,
    IMPORTANT_HIGH_SEVERITY_COLOR,
    MODERATE_MEDIUM_SEVERITY_COLOR,
    UNKNOWN_SEVERITY_COLOR,
} from 'constants/severityColors';
import type { VulnerabilitySeverity } from 'types/cve.proto';
import type { PolicySeverity } from 'types/policy.proto';

// Defines the default PF icons that represent a CVE severity, and sets the default colors for the icons.
// Color can be overridden by passing the standard `color` prop to the icon component.
// For example, colors={count === 0 ? noViolationsColor: undefined} prop.

export const CriticalSeverityIcon = ({ color, ...props }: SVGIconProps) => (
    <>
        <Icon>
            <CriticalRiskIcon color={color ?? CRITICAL_SEVERITY_COLOR} {...props} />
        </Icon>
    </>
);

export const ImportantSeverityIcon = ({ color, ...props }: SVGIconProps) => (
    <Icon>
        <AngleDoubleUpIcon color={color ?? IMPORTANT_HIGH_SEVERITY_COLOR} {...props} />
    </Icon>
);

export const HighSeverityIcon = ImportantSeverityIcon; // High is for policy severity

export const ModerateSeverityIcon = ({ color, ...props }: SVGIconProps) => (
    <Icon>
        <EqualsIcon color={color ?? MODERATE_MEDIUM_SEVERITY_COLOR} {...props} />
    </Icon>
);

export const MediumSeverityIcon = ModerateSeverityIcon; // Medium is for policy severity

export const LowSeverityIcon = ({ color, ...props }: SVGIconProps) => (
    <Icon>
        <AngleDoubleDownIcon color={color ?? LOW_SEVERITY_COLOR} {...props} />
    </Icon>
);

export const UnknownSeverityIcon = ({ color, ...props }: SVGIconProps) => (
    <Icon>
        <UnknownIcon color={color ?? UNKNOWN_SEVERITY_COLOR} {...props} />
    </Icon>
);

const SeverityIcons: Record<VulnerabilitySeverity, FC<PropsWithChildren<SVGIconProps>>> = {
    CRITICAL_VULNERABILITY_SEVERITY: CriticalSeverityIcon,
    IMPORTANT_VULNERABILITY_SEVERITY: ImportantSeverityIcon,
    MODERATE_VULNERABILITY_SEVERITY: ModerateSeverityIcon,
    LOW_VULNERABILITY_SEVERITY: LowSeverityIcon,
    UNKNOWN_VULNERABILITY_SEVERITY: UnknownSeverityIcon,
};

export const policySeverityIconMap: Record<PolicySeverity, FC<PropsWithChildren<SVGIconProps>>> = {
    CRITICAL_SEVERITY: CriticalSeverityIcon,
    HIGH_SEVERITY: HighSeverityIcon,
    MEDIUM_SEVERITY: MediumSeverityIcon,
    LOW_SEVERITY: LowSeverityIcon,
};

export default SeverityIcons;
