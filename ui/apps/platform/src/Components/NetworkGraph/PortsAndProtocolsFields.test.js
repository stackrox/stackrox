import { getPortsAndProtocolsMap, getPortsText } from './PortsAndProtocolsFields';

describe('getPortsAndProtocolsMap', () => {
    it('should group an array of ports/protocols where the key is a protocol and the value is a list of ports', () => {
        const data = [
            { port: 20, protocol: 'L4_PROTOCOL_TCP' },
            { port: 21, protocol: 'L4_PROTOCOL_TCP' },
            { port: 22, protocol: 'L4_PROTOCOL_TCP' },
            { port: 23, protocol: 'L4_PROTOCOL_TCP' },
            { port: 105, protocol: 'L4_PROTOCOL_UDP' },
            { port: 107, protocol: 'L4_PROTOCOL_UDP' },
        ];
        const expectedResult = {
            TCP: [20, 21, 22, 23],
            UDP: [105, 107],
        };

        const result = getPortsAndProtocolsMap(data);
        expect(result).toEqual(expectedResult);
    });
});

describe('getPortsText', () => {
    it('should properly display the text for 5 ports or less', () => {
        const data = [20, 21, 22, 23, 25];
        const expectedResult = '20, 21, 22, 23, 25';

        const result = getPortsText(data);
        expect(result).toEqual(expectedResult);
    });

    it('should add a "+N more" suffix when there are more than 5 ports', () => {
        const data = [20, 21, 22, 23, 25, 26, 27, 28];
        const expectedResult = '20, 21, 22, 23, 25, +3 more';

        const result = getPortsText(data);
        expect(result).toEqual(expectedResult);
    });
});
