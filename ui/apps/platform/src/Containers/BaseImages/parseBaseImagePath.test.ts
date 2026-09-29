import { describe, expect, it } from 'vitest';

import { getTagPatternError, parseBaseImagePath } from './BaseImagesModal';

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

    it('should keep the tag pattern separate for a registry with a port', () => {
        const result = parseBaseImagePath('registry:5000/library/ubuntu:1.*');

        expect(result).toEqual({
            repoPath: 'registry:5000/library/ubuntu',
            tagPattern: '1.*',
        });
    });

    it('should return an empty tag when a registry with a port has no tag', () => {
        const result = parseBaseImagePath('registry:5000/library/ubuntu');

        expect(result).toEqual({
            repoPath: 'registry:5000/library/ubuntu',
            tagPattern: '',
        });
    });

    it('should return an empty tag when the pattern is placed on the path instead of the tag', () => {
        const result = parseBaseImagePath('registry:5000/library/ubuntu*');

        expect(result).toEqual({
            repoPath: 'registry:5000/library/ubuntu*',
            tagPattern: '',
        });
    });
});

describe('getTagPatternError', () => {
    it('accepts a tag pattern at the length limit', () => {
        expect(getTagPatternError('a'.repeat(128))).toBeUndefined();
    });

    it('rejects tag patterns longer than 128 bytes', () => {
        expect(getTagPatternError('a'.repeat(129))).toBe('Tag pattern must be at most 128 bytes');
    });

    it('rejects repository separators and whitespace', () => {
        ['repo/tag', 'port:tag', 'sha256@digest', '1. *', '1.\u00a0*'].forEach((pattern) => {
            expect(getTagPatternError(pattern)).toBe(
                'Tag pattern must not contain "/", ":", "@", or whitespace'
            );
        });
    });
});
