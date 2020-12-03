import decodeBase64 from 'utils/decodeBase64';

describe('decodeBase64', () => {
    it('decodes a base64 encoded string', () => {
        const encodedValue = 'My4yMi4wLTF1YnVudHUwLjE';

        const decodedValue = decodeBase64(encodedValue);

        expect(decodedValue).toEqual('3.22.0-1ubuntu0.1');
    });

    it('decodes a base64 encoded string that contains a ~ character', () => {
        const encodedValue = 'My40LjMtMXVidW50dTF-MTQuMDQuNw';

        const decodedValue = decodeBase64(encodedValue);

        expect(decodedValue).toEqual('3.4.3-1ubuntu1~14.04.7');
    });
});
