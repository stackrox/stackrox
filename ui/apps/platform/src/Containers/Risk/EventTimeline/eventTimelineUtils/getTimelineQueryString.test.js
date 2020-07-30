import getTimelineQueryString from './getTimelineQueryString';

describe('getTimelineQueryString', () => {
    it('should a query string for a Deployment timeline', () => {
        const exportParams = {
            'Deployment ID': 'e053fa57-b70f-11ea-acbb-025000000001',
        };

        const str = getTimelineQueryString(exportParams);

        expect(str).toEqual('query=Deployment ID:"e053fa57-b70f-11ea-acbb-025000000001"');
    });

    it('should a query string for a Pod timeline', () => {
        const exportParams = {
            'Deployment ID': 'e053fa57-b70f-11ea-acbb-025000000001',
            'Pod ID': '49b84661-8915-5356-8876-bd222a86f779',
        };

        const str = getTimelineQueryString(exportParams);

        expect(str).toEqual(
            'query=Deployment ID:"e053fa57-b70f-11ea-acbb-025000000001"&Pod ID:"49b84661-8915-5356-8876-bd222a86f779"'
        );
    });
});
