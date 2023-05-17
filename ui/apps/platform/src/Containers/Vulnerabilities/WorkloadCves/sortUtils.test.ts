import { sortCveDistroList } from './sortUtils';

describe('sortCveDistroList', () => {
    it('should return an array of objects sorted by operating system priority', () => {
        const summaries = [{ operatingSystem: 'amzn:2' }];

        expect(sortCveDistroList(summaries)).toEqual([
            { distro: 'amzn', operatingSystem: 'amzn:2' },
        ]);

        summaries.push({ operatingSystem: 'alpine:v.3.2' });
        expect(sortCveDistroList(summaries)).toEqual([
            { distro: 'alpine', operatingSystem: 'alpine:v.3.2' },
            { distro: 'amzn', operatingSystem: 'amzn:2' },
        ]);

        summaries.push({ operatingSystem: 'amzn:2018.03' });
        expect(sortCveDistroList(summaries)).toEqual([
            { distro: 'alpine', operatingSystem: 'alpine:v.3.2' },
            { distro: 'amzn', operatingSystem: 'amzn:2' },
            { distro: 'amzn', operatingSystem: 'amzn:2018.03' },
        ]);

        // Add an "unknown" OS
        summaries.push({ operatingSystem: 'windows:xp' });
        expect(sortCveDistroList(summaries)).toEqual([
            { distro: 'alpine', operatingSystem: 'alpine:v.3.2' },
            { distro: 'amzn', operatingSystem: 'amzn:2' },
            { distro: 'amzn', operatingSystem: 'amzn:2018.03' },
            { distro: 'other', operatingSystem: 'windows:xp' },
        ]);

        summaries.push({ operatingSystem: 'debian:9' });
        expect(sortCveDistroList(summaries)).toEqual([
            { distro: 'debian', operatingSystem: 'debian:9' },
            { distro: 'alpine', operatingSystem: 'alpine:v.3.2' },
            { distro: 'amzn', operatingSystem: 'amzn:2' },
            { distro: 'amzn', operatingSystem: 'amzn:2018.03' },
            { distro: 'other', operatingSystem: 'windows:xp' },
        ]);

        summaries.push({ operatingSystem: 'ubuntu:20.04' });
        expect(sortCveDistroList(summaries)).toEqual([
            { distro: 'ubuntu', operatingSystem: 'ubuntu:20.04' },
            { distro: 'debian', operatingSystem: 'debian:9' },
            { distro: 'alpine', operatingSystem: 'alpine:v.3.2' },
            { distro: 'amzn', operatingSystem: 'amzn:2' },
            { distro: 'amzn', operatingSystem: 'amzn:2018.03' },
            { distro: 'other', operatingSystem: 'windows:xp' },
        ]);

        summaries.push({ operatingSystem: 'rhel:9' });
        expect(sortCveDistroList(summaries)).toEqual([
            { distro: 'rhel', operatingSystem: 'rhel:9' },
            { distro: 'ubuntu', operatingSystem: 'ubuntu:20.04' },
            { distro: 'debian', operatingSystem: 'debian:9' },
            { distro: 'alpine', operatingSystem: 'alpine:v.3.2' },
            { distro: 'amzn', operatingSystem: 'amzn:2' },
            { distro: 'amzn', operatingSystem: 'amzn:2018.03' },
            { distro: 'other', operatingSystem: 'windows:xp' },
        ]);

        summaries.push({ operatingSystem: 'centos:8' });
        expect(sortCveDistroList(summaries)).toEqual([
            { distro: 'rhel', operatingSystem: 'rhel:9' },
            { distro: 'centos', operatingSystem: 'centos:8' },
            { distro: 'ubuntu', operatingSystem: 'ubuntu:20.04' },
            { distro: 'debian', operatingSystem: 'debian:9' },
            { distro: 'alpine', operatingSystem: 'alpine:v.3.2' },
            { distro: 'amzn', operatingSystem: 'amzn:2' },
            { distro: 'amzn', operatingSystem: 'amzn:2018.03' },
            { distro: 'other', operatingSystem: 'windows:xp' },
        ]);
    });
});
