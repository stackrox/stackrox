import type { NotifierConfiguration } from 'services/ReportsService.types';
import type { Schedule } from 'types/schedule.proto';

export type ReportPageAction = 'clone' | 'create' | 'edit' | 'createFromFilters';

export type DetailsType = {
    name: string;
    description: string;
};

export type DeliveryType = {
    notifiers: NotifierConfiguration[];
    schedule: Schedule | null;
};
