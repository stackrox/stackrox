import entityTypes from 'constants/entityTypes';
import filterEntityRelationship from './filterEntityRelationship';

describe('filterEntityRelationship', () => {
    it('should return the given types for types not affected by VM updates', () => {
        const showVMUpdates = true;
        const values = [
            entityTypes.CLUSTER,
            entityTypes.NAMESPACE,
            entityTypes.DEPLOYMENT,
            entityTypes.IMAGE,
            entityTypes.NODE,
            entityTypes.POLICY,
        ];

        const filteredValues = values.filter((value) => {
            return filterEntityRelationship(showVMUpdates, value);
        });

        expect(filteredValues).toEqual(values);
    });

    it('should return only old types when the VM update flag is off', () => {
        const showVMUpdates = false;
        const values = [
            entityTypes.CLUSTER,
            entityTypes.NAMESPACE,
            entityTypes.DEPLOYMENT,
            entityTypes.IMAGE,
            entityTypes.COMPONENT,
            entityTypes.IMAGE_COMPONENT,
            entityTypes.NODE_COMPONENT,
            entityTypes.CVE,
            entityTypes.IMAGE_CVE,
            entityTypes.NODE_CVE,
            entityTypes.CLUSTER_CVE,
            entityTypes.NODE,
            entityTypes.POLICY,
        ];

        const filteredValues = values.filter((value) => {
            return filterEntityRelationship(showVMUpdates, value);
            // const foo = filterEntityRelationship(showVMUpdates, value);
            // return foo;
        });

        const expectedValues = [
            entityTypes.CLUSTER,
            entityTypes.NAMESPACE,
            entityTypes.DEPLOYMENT,
            entityTypes.IMAGE,
            entityTypes.COMPONENT,
            entityTypes.CVE,
            entityTypes.NODE,
            entityTypes.POLICY,
        ];
        expect(filteredValues).toEqual(expectedValues);
    });

    it('should return only new types when the VM update flag is on', () => {
        const showVMUpdates = true;
        const values = [
            entityTypes.CLUSTER,
            entityTypes.NAMESPACE,
            entityTypes.DEPLOYMENT,
            entityTypes.IMAGE,
            entityTypes.COMPONENT,
            entityTypes.IMAGE_COMPONENT,
            entityTypes.NODE_COMPONENT,
            entityTypes.CVE,
            entityTypes.IMAGE_CVE,
            entityTypes.NODE_CVE,
            entityTypes.CLUSTER_CVE,
            entityTypes.NODE,
            entityTypes.POLICY,
        ];

        const filteredValues = values.filter((value) => {
            return filterEntityRelationship(showVMUpdates, value);
            // const foo = filterEntityRelationship(showVMUpdates, value);
            // return foo;
        });

        const expectedValues = [
            entityTypes.CLUSTER,
            entityTypes.NAMESPACE,
            entityTypes.DEPLOYMENT,
            entityTypes.IMAGE,
            entityTypes.IMAGE_COMPONENT,
            entityTypes.NODE_COMPONENT,
            entityTypes.IMAGE_CVE,
            entityTypes.NODE_CVE,
            entityTypes.CLUSTER_CVE,
            entityTypes.NODE,
            entityTypes.POLICY,
        ];
        expect(filteredValues).toEqual(expectedValues);
    });
});
