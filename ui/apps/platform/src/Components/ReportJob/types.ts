export const jobContextTabs = ['CONFIGURATION_DETAILS', 'ALL_REPORT_JOBS'] as const;

export type JobContextTab = (typeof jobContextTabs)[number];
