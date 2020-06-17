import { getRatioOfScannedImages } from './deployments.utils';

describe('Vuln Mgmt Deployments list utils', () => {
    describe('getRatioOfScannedImages', () => {
        it('should return 0/0 when an empty array is passed in', () => {
            const images = [];

            const scanRatio = getRatioOfScannedImages(images);

            expect(scanRatio).toEqual({ scanned: 0, total: 0 });
        });

        it('should return 0/1 when an array with single item with no single scan is passed in', () => {
            const images = [
                {
                    scan: {
                        scanTime: null,
                    },
                },
            ];

            const scanRatio = getRatioOfScannedImages(images);

            expect(scanRatio).toEqual({ scanned: 0, total: 1 });
        });

        it('should return 1/1 when an array with a single scan is passed in', () => {
            const images = [
                {
                    scan: {
                        scanTime: '2020-06-12T17:43:27.0865805Z',
                    },
                },
            ];

            const scanRatio = getRatioOfScannedImages(images);

            expect(scanRatio).toEqual({ scanned: 1, total: 1 });
        });

        it('should return an equal ratio when an array with all items scanned', () => {
            const images = [
                {
                    scan: {
                        scanTime: '2020-06-12T17:43:27.0865805Z',
                    },
                },
                {
                    scan: {
                        scanTime: '2020-06-12T17:51:27.08549105Z',
                    },
                },
                {
                    scan: {
                        scanTime: '2020-06-12T16:01:50.0865805Z',
                    },
                },
            ];

            const scanRatio = getRatioOfScannedImages(images);

            expect(scanRatio).toEqual({ scanned: 3, total: 3 });
        });

        it('should return the ratio when not all items are scanned', () => {
            const images = [
                {
                    scan: {
                        scanTime: '2020-06-12T17:43:27.0865805Z',
                    },
                },
                {
                    scan: {
                        scanTime: '',
                    },
                },
                {
                    scan: {
                        scanTime: '2020-06-12T16:01:50.0865805Z',
                    },
                },
            ];

            const scanRatio = getRatioOfScannedImages(images);

            expect(scanRatio).toEqual({ scanned: 2, total: 3 });
        });
    });
});
