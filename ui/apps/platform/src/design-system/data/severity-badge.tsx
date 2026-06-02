import { Badge, type BadgeProps } from 'design-system/ui/badge';

type SeverityLevel = 'CRITICAL_SEVERITY' | 'HIGH_SEVERITY' | 'MEDIUM_SEVERITY' | 'LOW_SEVERITY';

const severityToVariant: Record<SeverityLevel, BadgeProps['variant']> = {
    CRITICAL_SEVERITY: 'critical',
    HIGH_SEVERITY: 'high',
    MEDIUM_SEVERITY: 'medium',
    LOW_SEVERITY: 'low',
};

const severityLabel: Record<SeverityLevel, string> = {
    CRITICAL_SEVERITY: 'Critical',
    HIGH_SEVERITY: 'High',
    MEDIUM_SEVERITY: 'Medium',
    LOW_SEVERITY: 'Low',
};

interface SeverityBadgeProps {
    severity: SeverityLevel;
    className?: string;
}

export function SeverityBadge({ severity, className }: SeverityBadgeProps) {
    return (
        <Badge variant={severityToVariant[severity]} className={className}>
            {severityLabel[severity]}
        </Badge>
    );
}
