import getRowColorStylesByStatus from './getRowColorStylesByStatus';

describe('getRowColorStylesByStatus', () => {
    it('should return the color styles for the added baseline', () => {
        expect(getRowColorStylesByStatus('ADDED')).toEqual({
            bgColor: 'bg-success-200',
            borderColor: 'border-success-300',
            textColor: 'text-success-800',
        });
    });

    it('should return the color styles for the removed baseline', () => {
        expect(getRowColorStylesByStatus('REMOVED')).toEqual({
            bgColor: 'bg-alert-200',
            borderColor: 'border-alert-300',
            textColor: 'text-alert-800',
        });
    });

    it('should return the color styles for the modified baseline', () => {
        expect(getRowColorStylesByStatus('MODIFIED')).toEqual({
            bgColor: 'bg-warning-200',
            borderColor: 'border-warning-300',
            textColor: 'text-warning-800',
        });
    });

    it('should return the color styles for the unmodified baseline', () => {
        expect(getRowColorStylesByStatus('UNMODIFIED')).toEqual({
            bgColor: 'bg-base-100',
            borderColor: 'border-base-300',
            textColor: '',
        });
    });
});
