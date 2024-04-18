import { Fixability } from 'services/ReportsService.types';

export type FixabilityLabelKey = Exclude<Fixability, 'BOTH'>;
type FixabilityLabels = Record<FixabilityLabelKey, string>;

export const fixabilityLabels: FixabilityLabels = {
    FIXABLE: 'Fixable',
    NOT_FIXABLE: 'Unfixable',
};
