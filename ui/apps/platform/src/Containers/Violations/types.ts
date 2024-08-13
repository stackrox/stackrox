export const violationStateTabs = ['ACTIVE', 'RESOLVED'] as const;

export type ViolationStateTab = (typeof violationStateTabs)[number];
