import type { NotifierConfiguration } from 'services/ReportsService.types';
import type { Schedule } from 'types/schedule.proto';

export type DeliveryType = {
    notifiers: NotifierConfiguration[];
    schedule: Schedule | null;
};

export type DetailsType = {
    name: string;
    description: string;
};
