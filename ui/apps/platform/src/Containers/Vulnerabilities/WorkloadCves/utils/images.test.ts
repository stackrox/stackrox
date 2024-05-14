import { getImageBaseNameDisplay } from './images';

describe('image utils', () => {
    describe('getImageBaseNameDisplay', () => {
        it('should return remote:tag when tag is provided', () => {
            const id = 'id';
            const name = {
                remote: 'remote',
                registry: 'registry',
                tag: 'tag',
            };
            expect(getImageBaseNameDisplay(id, name)).toEqual('remote:tag');
        });

        it('should return remote@id when tag is not provided', () => {
            const id = 'id';
            const name = {
                remote: 'remote',
                registry: 'registry',
                tag: '',
            };
            expect(getImageBaseNameDisplay(id, name)).toEqual('remote@id');
        });
    });
});
