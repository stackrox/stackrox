import type { Cluster, SensorVersionCompatibility } from 'types/cluster.proto';

import { getSensorCompatibilityCounts } from './SensorCompatibilityCard';

function clusterWithCompatibility(sensorVersionCompatibility: SensorVersionCompatibility): Cluster {
    return { status: { sensorVersionCompatibility } } as Cluster;
}

describe('getSensorCompatibilityCounts', () => {
    it('returns zero counts for no clusters', () => {
        expect(getSensorCompatibilityCounts([])).toEqual({
            MATCHED: 0,
            COMPATIBLE: 0,
            INCOMPATIBLE: 0,
            UNKNOWN: 0,
        });
    });

    // Both BEHIND and AHEAD variants collapse into a single bucket, so a new
    // enum case added without a matching test would surface here.
    it.each([
        ['SENSOR_VERSION_COMPATIBILITY_MATCHED', 'MATCHED'],
        ['SENSOR_VERSION_COMPATIBILITY_COMPATIBLE_BEHIND', 'COMPATIBLE'],
        ['SENSOR_VERSION_COMPATIBILITY_COMPATIBLE_AHEAD', 'COMPATIBLE'],
        ['SENSOR_VERSION_COMPATIBILITY_INCOMPATIBLE_BEHIND', 'INCOMPATIBLE'],
        ['SENSOR_VERSION_COMPATIBILITY_INCOMPATIBLE_AHEAD', 'INCOMPATIBLE'],
        ['SENSOR_VERSION_COMPATIBILITY_UNKNOWN', 'UNKNOWN'],
    ] as const)('counts %s as %s', (compatibility, bucket) => {
        const counts = getSensorCompatibilityCounts([clusterWithCompatibility(compatibility)]);

        expect(counts[bucket]).toBe(1);
        // Every other bucket stays at zero.
        expect(Object.values(counts).reduce((sum, value) => sum + value, 0)).toBe(1);
    });

    it('counts a cluster without status as unknown', () => {
        const counts = getSensorCompatibilityCounts([{} as Cluster]);

        expect(counts).toEqual({
            MATCHED: 0,
            COMPATIBLE: 0,
            INCOMPATIBLE: 0,
            UNKNOWN: 1,
        });
    });

    it('tallies a mix of compatibility states across clusters', () => {
        const clusters = [
            clusterWithCompatibility('SENSOR_VERSION_COMPATIBILITY_MATCHED'),
            clusterWithCompatibility('SENSOR_VERSION_COMPATIBILITY_MATCHED'),
            clusterWithCompatibility('SENSOR_VERSION_COMPATIBILITY_COMPATIBLE_AHEAD'),
            clusterWithCompatibility('SENSOR_VERSION_COMPATIBILITY_INCOMPATIBLE_BEHIND'),
            clusterWithCompatibility('SENSOR_VERSION_COMPATIBILITY_UNKNOWN'),
            {} as Cluster,
        ];

        expect(getSensorCompatibilityCounts(clusters)).toEqual({
            MATCHED: 2,
            COMPATIBLE: 1,
            INCOMPATIBLE: 1,
            UNKNOWN: 2,
        });
    });
});
