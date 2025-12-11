import { describe, expect, it } from 'vitest';

import { parseBaseImagePath } from './BaseImagesModal';

describe('parseBaseImagePath', () => {
    it('should parse simple image path with single colon', () => {
        const result = parseBaseImagePath('ubuntu:22.04');

        expect(result).toEqual({
            repoPath: 'ubuntu',
            tagPattern: '22.04',
        });
    });

    it('should parse image path with multiple colons (registry with port)', () => {
        const result = parseBaseImagePath('docker.io:5000/library/ubuntu:22.04');

        expect(result).toEqual({
            repoPath: 'docker.io:5000/library/ubuntu',
            tagPattern: '22.04',
        });
    });

    it('should parse image path with tag pattern', () => {
        const result = parseBaseImagePath('docker.io/library/ubuntu:1.*');

        expect(result).toEqual({
            repoPath: 'docker.io/library/ubuntu',
            tagPattern: '1.*',
        });
    });
});
