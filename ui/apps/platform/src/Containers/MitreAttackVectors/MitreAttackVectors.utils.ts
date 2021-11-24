const mitreAttackUrl = 'https://attack.mitre.org';

/*
 * Return …/tactics/TA0001/ for tactic TA0001 Initial Access
 */
export function getMitreTacticUrl(id: string): string {
    return `${mitreAttackUrl}/tactics/${id}/`;
}

/*
 * Return …/techniques/T1566/ for technique T1566 Phishing
 * Return …/techniques/T1566/001/ for sub-technique T1566.001 Spearphishing Attachment
 */
export function getMitreTechniqueUrl(id: string): string {
    return `${mitreAttackUrl}/techniques/${id.replace('.', '/')}/`;
}
