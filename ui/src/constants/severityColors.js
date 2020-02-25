//  @TODO: Have one source of truth for severity colors
export const severityColorMap = {
    CRITICAL_SEVERITY: 'var(--alert-400)',
    HIGH_SEVERITY: 'var(--caution-400)',
    MEDIUM_SEVERITY: 'var(--warning-400)',
    LOW_SEVERITY: 'var(--tertiary-400)'
};

export const severityTextColorMap = {
    CRITICAL_SEVERITY: 'var(--alert-700)',
    HIGH_SEVERITY: 'var(--caution-700)',
    MEDIUM_SEVERITY: 'var(--warning-700)',
    LOW_SEVERITY: 'var(--tertiary-700)'
};

export const severityColorLegend = [
    {
        title: 'Low',
        color: severityColorMap.LOW_SEVERITY,
        textColor: severityTextColorMap.LOW_SEVERITY
    },
    {
        title: 'Medium',
        color: severityColorMap.MEDIUM_SEVERITY,
        textColor: severityTextColorMap.MEDIUM_SEVERITY
    },
    {
        title: 'High',
        color: severityColorMap.HIGH_SEVERITY,
        textColor: severityTextColorMap.HIGH_SEVERITY
    },
    {
        title: 'Critical',
        color: severityColorMap.CRITICAL_SEVERITY,
        textColor: severityTextColorMap.CRITICAL_SEVERITY
    }
];

export default {
    severityColorMap,
    severityColorLegend
};
