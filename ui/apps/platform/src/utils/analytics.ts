export function analyticsIdentity(userId: string, traits = {}): void {
    return window.analytics?.identify(userId, traits);
}

export function analyticsPageVisit(type: string, name: string, additionalProperties = {}): void {
    return window.analytics?.page(type, name, additionalProperties);
}

export function analyticsTrack(event: string, additionalProperties = {}): void {
    return window.analytics?.track(event, additionalProperties);
}

export const clusterCreated = 'Cluster Created';
