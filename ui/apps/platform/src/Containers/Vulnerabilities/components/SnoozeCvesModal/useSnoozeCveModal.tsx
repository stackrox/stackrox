import { useState } from 'react';

export type SnoozeableCveType = 'CLUSTER_CVE' | 'NODE_CVE';
export type SnoozeAction = 'SNOOZE' | 'UNSNOOZE';

export default function useSnoozeCvesModal() {
    const [snoozeModalOptions, setSnoozeModalOptions] = useState<{
        action: SnoozeAction;
        cveType: SnoozeableCveType;
        cves: { cve: string }[];
    } | null>(null);

    function snoozeActionCreator(cveType: SnoozeableCveType, action: SnoozeAction) {
        const title = action === 'SNOOZE' ? 'Snooze CVE' : 'Unsnooze CVE';
        return ({ cve }: { cve: string }) => {
            const onClick = () => {
                setSnoozeModalOptions({
                    action,
                    cveType,
                    cves: [{ cve }],
                });
            };

            return [{ title, onClick }];
        };
    }

    return {
        snoozeModalOptions,
        setSnoozeModalOptions,
        snoozeActionCreator,
    };
}
