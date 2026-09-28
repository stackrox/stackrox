import { computeVersionRangeData } from './SensorVersionRangeChart';

// Central is 5.0, +3/-3 compatible versions
const versions = ['4.7', '4.8', '4.9', '5.0', '5.1', '5.2', '5.3'];
const centralVersion = '5.0.x-nightly';
const centralIndex = 3;
const totalUnits = versions.length + 2; // 9

describe('computeVersionRangeData', () => {
    describe('null returns for invalid inputs', () => {
        it('returns null when compatible versions list is empty', () => {
            expect(computeVersionRangeData([], '4.9.0', centralVersion, undefined)).toBeNull();
        });

        it('returns null when central version is not parseable', () => {
            expect(computeVersionRangeData(versions, '4.9.0', 'latest', undefined)).toBeNull();
        });

        it('returns null when central X.Y is not in compatible versions', () => {
            expect(computeVersionRangeData(versions, '4.9.0', '3.0.0', undefined)).toBeNull();
        });
    });

    describe('zone structure', () => {
        it('produces incompatible bookends, compatible behind/ahead, and matched zones', () => {
            const result = computeVersionRangeData(
                versions,
                '4.8.0',
                centralVersion,
                'SENSOR_VERSION_COMPATIBILITY_COMPATIBLE_BEHIND'
            );

            const colors = result!.zones.map((z) => z.color);
            // incompatible | compatible-behind | matched | compatible-ahead | incompatible
            expect(colors).toEqual([
                expect.stringContaining('danger'),
                expect.stringContaining('info'),
                expect.stringContaining('success'),
                expect.stringContaining('info'),
                expect.stringContaining('danger'),
            ]);
        });

        it('zone widths sum to totalUnits (versions.length + 2)', () => {
            const result = computeVersionRangeData(
                versions,
                '5.0.0',
                centralVersion,
                'SENSOR_VERSION_COMPATIBILITY_MATCHED'
            );

            const totalWidth = result!.zones.reduce((sum, z) => sum + z.width, 0);
            expect(totalWidth).toBe(totalUnits);
        });
    });

    describe('marker positioning', () => {
        it('places marker at the parsed sensor position when behind', () => {
            // Sensor 4.8 is at index 1, behind central 5.0 at index 3
            const sensor48Index = 1;
            const result = computeVersionRangeData(
                versions,
                '4.8.0',
                centralVersion,
                'SENSOR_VERSION_COMPATIBILITY_COMPATIBLE_BEHIND'
            );

            const expected = ((1 + sensor48Index + 0.5) / totalUnits) * 100;
            expect(result!.markerPercent).toBeCloseTo(expected);
        });

        it('places marker at matched position when sensor equals central', () => {
            const result = computeVersionRangeData(
                versions,
                '5.0.0',
                centralVersion,
                'SENSOR_VERSION_COMPATIBILITY_MATCHED'
            );

            const expected = ((1 + centralIndex + 0.5) / totalUnits) * 100;
            expect(result!.markerPercent).toBeCloseTo(expected);
        });

        it('places marker at parsed sensor position when ahead', () => {
            // Sensor 5.2 is at index 5, ahead of central 5.0 at index 3
            const sensor52Index = 5;
            const result = computeVersionRangeData(
                versions,
                '5.2.0',
                centralVersion,
                'SENSOR_VERSION_COMPATIBILITY_COMPATIBLE_AHEAD'
            );

            const expected = ((1 + sensor52Index + 0.5) / totalUnits) * 100;
            expect(result!.markerPercent).toBeCloseTo(expected);
        });
    });

    describe('marker fallback when sensor version is not in the compatible list', () => {
        it('falls back to the left edge for behind compatibility', () => {
            const result = computeVersionRangeData(
                versions,
                '3.0.0',
                centralVersion,
                'SENSOR_VERSION_COMPATIBILITY_INCOMPATIBLE_BEHIND'
            );

            const expected = (0.5 / totalUnits) * 100;
            expect(result!.markerPercent).toBeCloseTo(expected);
        });

        it('falls back to the right edge for ahead compatibility', () => {
            const result = computeVersionRangeData(
                versions,
                '99.0.0',
                centralVersion,
                'SENSOR_VERSION_COMPATIBILITY_INCOMPATIBLE_AHEAD'
            );

            const expected = ((totalUnits - 0.5) / totalUnits) * 100;
            expect(result!.markerPercent).toBeCloseTo(expected);
        });

        it('falls back to central position for matched compatibility with unparseable sensor', () => {
            const result = computeVersionRangeData(
                versions,
                'unparseable',
                centralVersion,
                'SENSOR_VERSION_COMPATIBILITY_MATCHED'
            );

            const expected = ((1 + centralIndex + 0.5) / totalUnits) * 100;
            expect(result!.markerPercent).toBeCloseTo(expected);
        });

        it('returns null marker when both compatibility is unknown and sensor version is unparseable', () => {
            const result = computeVersionRangeData(
                versions,
                'unparseable',
                centralVersion,
                'SENSOR_VERSION_COMPATIBILITY_UNKNOWN'
            );

            expect(result!.markerPercent).toBeNull();
        });

        // If the sensor version IS parseable, we show the marker even with unknown compatibility
        it('still shows marker for unknown compatibility when sensor version is parseable', () => {
            const result = computeVersionRangeData(
                versions,
                '5.0.0',
                centralVersion,
                'SENSOR_VERSION_COMPATIBILITY_UNKNOWN'
            );

            expect(result!.markerPercent).not.toBeNull();
        });
    });

    describe('marker/compatibility contradiction', () => {
        // Sensor 4.8 is at index 1, which is behind central (index 3),
        // but the backend says "ahead". The parsed position contradicts
        // the enum, so we discard it and fall back to the enum-based
        // position: the middle of the compatible-ahead zone.
        it('discards the parsed position and falls back when it contradicts the compatibility enum', () => {
            const result = computeVersionRangeData(
                versions,
                '4.8.0',
                centralVersion,
                'SENSOR_VERSION_COMPATIBILITY_COMPATIBLE_AHEAD'
            );

            const aheadCount = versions.length - centralIndex - 1;
            const fallbackAhead = ((1 + centralIndex + 1 + aheadCount / 2) / totalUnits) * 100;
            expect(result!.markerPercent).toBeCloseTo(fallbackAhead);
        });
    });
});
