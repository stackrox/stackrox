import { Schedule } from 'services/ComplianceScanConfigurationService';
import {
    convertFormikParametersToSchedule,
    convertScheduleToFormikParameters,
    ScanConfigParameters,
} from './compliance.scanConfigs.utils';

// @TODO: Consider making a more unique name for general utils file under Vulnerability Reporting
describe('compliance.scanConfigs.utils', () => {
    describe('convertFormikParametersToSchedule', () => {
        it('should return the correct Daily Scan Schedule for the given daily formik values', () => {
            const formValues: ScanConfigParameters = {
                name: 'ok-ok.ok',
                description: 'Needles and Pins',
                intervalType: 'DAILY',
                time: '03:00',
                daysOfWeek: [],
                daysOfMonth: [],
            };

            const scanConfig = convertFormikParametersToSchedule(formValues);

            expect(scanConfig).toEqual({
                hour: 3,
                minute: 0,
                intervalType: 'DAILY',
            });
        });

        it('should return the correct Weekly Scan Schedule for the given weekly formik values', () => {
            const formValues: ScanConfigParameters = {
                name: 'once-a-week',
                description:
                    'Several Species of Small Furry Animals Gathered Together in a Cave and Grooving with a Pict',
                intervalType: 'WEEKLY',
                time: '13:00',
                daysOfWeek: ['1'],
                daysOfMonth: [],
            };

            const scanConfig = convertFormikParametersToSchedule(formValues);

            expect(scanConfig).toEqual({
                hour: 13,
                minute: 0,
                intervalType: 'WEEKLY',
                daysOfWeek: {
                    days: [1],
                },
            });
        });

        it('should return the correct Monthly Scan Schedule for the given monthly formik values', () => {
            const formValues: ScanConfigParameters = {
                name: 'once-a-week',
                description:
                    'Several Species of Small Furry Animals Gathered Together in a Cave and Grooving with a Pict',
                intervalType: 'MONTHLY',
                time: '23:00',
                daysOfWeek: [],
                daysOfMonth: ['1', '15'],
            };

            const scanConfig = convertFormikParametersToSchedule(formValues);

            expect(scanConfig).toEqual({
                hour: 23,
                minute: 0,
                intervalType: 'MONTHLY',
                daysOfMonth: {
                    days: [1, 15],
                },
            });
        });
    });

    describe('convertScheduleToFormikParameters', () => {
        it('should return the correct daily formik values for the given Daily Scan Schedule', () => {
            const scanSchedule: Schedule = {
                hour: 22,
                minute: 0,
                intervalType: 'DAILY',
            };

            const formValues = convertScheduleToFormikParameters(scanSchedule);

            expect(formValues).toEqual({
                intervalType: 'DAILY',
                time: '22:00',
                daysOfWeek: [],
                daysOfMonth: [],
            });
        });

        it('should return the correct weekly formik values for the given Weekly Scan Schedule', () => {
            const scanSchedule: Schedule = {
                hour: 15,
                minute: 0,
                intervalType: 'WEEKLY',
                daysOfWeek: {
                    days: [1],
                },
            };

            const formValues = convertScheduleToFormikParameters(scanSchedule);

            expect(formValues).toEqual({
                intervalType: 'WEEKLY',
                time: '15:00',
                daysOfWeek: ['1'],
                daysOfMonth: [],
            });
        });

        it('should return the correct monthly formik values for the given Monthly Scan Schedule', () => {
            const scanSchedule: Schedule = {
                hour: 5,
                minute: 0,
                intervalType: 'MONTHLY',
                daysOfMonth: {
                    days: [15],
                },
            };

            const formValues = convertScheduleToFormikParameters(scanSchedule);

            expect(formValues).toEqual({
                intervalType: 'MONTHLY',
                time: '05:00',
                daysOfWeek: [],
                daysOfMonth: ['15'],
            });
        });
    });
});
