export const violationStateTabs = ['ACTIVE', 'RESOLVED', 'ATTEMPTED'] as const;

export type ViolationStateTab = (typeof violationStateTabs)[number];
