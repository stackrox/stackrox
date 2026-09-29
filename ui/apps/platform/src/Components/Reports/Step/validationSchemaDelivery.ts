import * as yup from 'yup';

import type { Schedule } from 'types/schedule.proto';

import type { DeliveryType } from '../reports.types';

export const validationSchemaDelivery: yup.ObjectSchema<DeliveryType> = yup.object().shape({
    notifiers: yup
        .array()
        .of(
            yup.object().shape({
                emailConfig: yup.object().shape({
                    notifierId: yup.string().required('Email notifier is required'),
                    mailingLists: yup
                        .array()
                        .of(yup.string().required())
                        .min(1, 'At least one distribution list email is required')
                        .required(),
                    customSubject: yup.string().defined(),
                    customBody: yup.string().defined(),
                }),
                notifierName: yup.string().defined(),
            })
        )
        .defined(),
    schedule: yup
        .object()
        .nullable()
        .defined()
        .test('schedule-days-required', 'At least one day must be selected', function test(value) {
            if (value == null) {
                return true;
            }
            // Schedule is a union type that .shape() can't express.
            // Cast so property access is type-checked.
            const schedule = value as Schedule;
            if (schedule.intervalType === 'UNSET' || schedule.intervalType === 'DAILY') {
                return true;
            }
            if ('daysOfWeek' in schedule && schedule.daysOfWeek.days.length === 0) {
                return this.createError({
                    path: 'schedule.daysOfWeek.days',
                    message: 'At least one day must be selected',
                });
            }
            if ('daysOfMonth' in schedule && schedule.daysOfMonth.days.length === 0) {
                return this.createError({
                    path: 'schedule.daysOfMonth.days',
                    message: 'At least one day must be selected',
                });
            }
            return true;
        }) as unknown as yup.Schema<Schedule | null>,
});
