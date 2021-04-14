import getDirectionalityLabel from './getDirectionalityLabel';

describe('getDirectionalityLabel', () => {
    it('should return the value "Ingress"', () => {
        const isIngress = true;
        expect(getDirectionalityLabel(isIngress)).toEqual('Ingress');
    });

    it('should return the value "Egress"', () => {
        const isIngress = false;
        expect(getDirectionalityLabel(isIngress)).toEqual('Egress');
    });
});
