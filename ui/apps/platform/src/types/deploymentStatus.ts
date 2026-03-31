export const deploymentStatuses = ['DEPLOYED', 'DELETED'] as const;
export type DeploymentStatus = (typeof deploymentStatuses)[number];

export function isDeploymentStatus(value: unknown): value is DeploymentStatus {
    return deploymentStatuses.some((s) => s === value);
}
