import {
    areNodeRolesValid,
    convertFormikParametersToSchedule,
    convertScanConfigToFormik,
    convertScheduleToFormikParameters,
    defaultNodeRoles,
    isValidNodeRole,
} from './compliance.scanConfigs.utils';
import type { ScanConfigParameters } from './compliance.scanConfigs.utils';

import type { ComplianceScanConfigurationStatus } from 'services/ComplianceScanConfigurationService';
import type { Schedule } from 'types/schedule.proto';

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
                nodeRoles: ['master', 'worker'],
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
                nodeRoles: ['master', 'worker'],
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
                nodeRoles: ['master', 'worker'],
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

    describe('convertScanConfigToFormik', () => {
        function makeExistingConfig(
            nodeRoles: string[] | undefined
        ): ComplianceScanConfigurationStatus {
            return {
                id: 'config-id',
                scanName: 'legacy-config',
                scanConfig: {
                    oneTimeScan: false,
                    profiles: ['ocp4-cis'],
                    scanSchedule: { hour: 3, minute: 0, intervalType: 'DAILY' },
                    description: '',
                    notifiers: [],
                    nodeRoles,
                },
                clusterStatus: [],
            } as unknown as ComplianceScanConfigurationStatus;
        }

        it('falls back to default node roles when a legacy config has empty node roles', () => {
            const formValues = convertScanConfigToFormik(makeExistingConfig([]));

            expect(formValues.parameters.nodeRoles).toEqual(defaultNodeRoles);
        });

        it('falls back to default node roles when node roles are missing', () => {
            const formValues = convertScanConfigToFormik(makeExistingConfig(undefined));

            expect(formValues.parameters.nodeRoles).toEqual(defaultNodeRoles);
        });

        it('passes custom node roles through unchanged', () => {
            const formValues = convertScanConfigToFormik(makeExistingConfig(['infra']));

            expect(formValues.parameters.nodeRoles).toEqual(['infra']);
        });

        it('returns a copy of the default node roles, not the shared reference', () => {
            const formValues = convertScanConfigToFormik(makeExistingConfig([]));

            expect(formValues.parameters.nodeRoles).toEqual(defaultNodeRoles);
            expect(formValues.parameters.nodeRoles).not.toBe(defaultNodeRoles);
        });
    });

    describe('isValidNodeRole', () => {
        it('accepts a concrete role and the @all wildcard', () => {
            expect(isValidNodeRole('master')).toBe(true);
            expect(isValidNodeRole('control-plane')).toBe(true);
            expect(isValidNodeRole('@all')).toBe(true);
        });

        it('rejects roles that are empty, too long, or have invalid characters', () => {
            expect(isValidNodeRole('')).toBe(false);
            expect(isValidNodeRole('bad role!')).toBe(false);
            expect(isValidNodeRole('a'.repeat(40))).toBe(false);
        });
    });

    describe('areNodeRolesValid', () => {
        it('accepts a valid array of concrete roles', () => {
            expect(areNodeRolesValid(['master', 'worker'])).toBe(true);
        });

        it('accepts @all on its own', () => {
            expect(areNodeRolesValid(['@all'])).toBe(true);
        });

        it('rejects an array containing an invalid role', () => {
            expect(areNodeRolesValid(['master', 'bad role!'])).toBe(false);
        });

        it('rejects @all combined with any other role', () => {
            expect(areNodeRolesValid(['@all', 'infra'])).toBe(false);
        });
    });
});
