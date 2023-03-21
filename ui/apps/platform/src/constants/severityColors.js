//  @TODO: Have one source of truth for severity colors
export const severityColorMap = {
    CRITICAL_SEVERITY: 'var(--alert-400)',
    HIGH_SEVERITY: 'var(--caution-400)',
    MEDIUM_SEVERITY: 'var(--warning-400)',
    LOW_SEVERITY: 'var(--tertiary-400)',
};

// color mapping of severities for patternfly labels
export const severityColorMapPF = {
    Low: 'gray',
    Medium: 'orange',
    High: 'red',
    Critical: 'purple',
};

// color mapping of vulnerability severities for patternfly labels
export const vulnerabilitySeverityColorMapPF = {
    Low: 'gray',
    Moderate: 'orange',
    Important: 'red',
    Critical: 'purple',
    Unknown: 'gray',
};

export const severityTextColorMap = {
    CRITICAL_SEVERITY: 'var(--alert-700)',
    HIGH_SEVERITY: 'var(--caution-700)',
    MEDIUM_SEVERITY: 'var(--warning-700)',
    LOW_SEVERITY: 'var(--tertiary-700)',
};

export const severityColorLegend = [
    {
        title: 'Low',
        color: severityColorMap.LOW_SEVERITY,
        textColor: severityTextColorMap.LOW_SEVERITY,
    },
    {
        title: 'Medium',
        color: severityColorMap.MEDIUM_SEVERITY,
        textColor: severityTextColorMap.MEDIUM_SEVERITY,
    },
    {
        title: 'High',
        color: severityColorMap.HIGH_SEVERITY,
        textColor: severityTextColorMap.HIGH_SEVERITY,
    },
    {
        title: 'Critical',
        color: severityColorMap.CRITICAL_SEVERITY,
        textColor: severityTextColorMap.CRITICAL_SEVERITY,
    },
];

export const cvssSeverityColorMap = {
    CRITICAL_VULNERABILITY_SEVERITY: 'var(--alert-400)',
    IMPORTANT_VULNERABILITY_SEVERITY: 'var(--caution-400)',
    MODERATE_VULNERABILITY_SEVERITY: 'var(--warning-400)',
    LOW_VULNERABILITY_SEVERITY: 'var(--tertiary-400)',
};

export const cvssSeverityTextColorMap = {
    CRITICAL_VULNERABILITY_SEVERITY: 'var(--alert-700)',
    IMPORTANT_VULNERABILITY_SEVERITY: 'var(--caution-700)',
    MODERATE_VULNERABILITY_SEVERITY: 'var(--warning-700)',
    LOW_VULNERABILITY_SEVERITY: 'var(--tertiary-700)',
};

export const cvssSeverityColorLegend = [
    {
        title: 'Low',
        color: cvssSeverityColorMap.LOW_VULNERABILITY_SEVERITY,
        textColor: cvssSeverityTextColorMap.LOW_VULNERABILITY_SEVERITY,
    },
    {
        title: 'Moderate',
        color: cvssSeverityColorMap.MODERATE_VULNERABILITY_SEVERITY,
        textColor: cvssSeverityTextColorMap.MODERATE_VULNERABILITY_SEVERITY,
    },
    {
        title: 'Important',
        color: cvssSeverityColorMap.IMPORTANT_VULNERABILITY_SEVERITY,
        textColor: cvssSeverityTextColorMap.IMPORTANT_VULNERABILITY_SEVERITY,
    },
    {
        title: 'Critical',
        color: cvssSeverityColorMap.CRITICAL_VULNERABILITY_SEVERITY,
        textColor: cvssSeverityTextColorMap.CRITICAL_VULNERABILITY_SEVERITY,
    },
];

export default {
    severityColorMap,
    severityColorLegend,
    cvssSeverityColorMap,
    cvssSeverityColorLegend,
};
