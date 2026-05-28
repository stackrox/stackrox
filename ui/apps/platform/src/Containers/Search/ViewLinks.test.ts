import { buildViewLinkUrl, resolveParams } from './ViewLinks';

describe('resolveParams', () => {
    it('should replace a single :param', () => {
        expect(resolveParams('s=:name', { name: 'nginx' })).toBe('s=nginx');
    });

    it('should replace multiple :params', () => {
        expect(
            resolveParams('ns=:name&cluster=:locationTextForCategory', {
                name: 'default',
                locationTextForCategory: 'remote',
            })
        ).toBe('ns=default&cluster=remote');
    });

    it('should preserve unmatched :params', () => {
        expect(resolveParams('s=:missing', {})).toBe('s=:missing');
    });

    it('should return string unchanged when no :params present', () => {
        expect(resolveParams('filteredWorkflowView=Full view', { id: '123' })).toBe(
            'filteredWorkflowView=Full view'
        );
    });
});

describe('buildViewLinkUrl', () => {
    it('should resolve :id in a simple path', () => {
        expect(buildViewLinkUrl('/main/risk/:id', { id: 'abc123' }, undefined)).toBe(
            '/main/risk/abc123'
        );
    });

    it('should append searchParams to a simple path', () => {
        expect(
            buildViewLinkUrl('/main/risk/:id', { id: 'abc123' }, 'filteredWorkflowView=Full view')
        ).toBe('/main/risk/abc123?filteredWorkflowView=Full view');
    });

    it('should resolve :params in basePath query string separately from path', () => {
        const basePath = '/main/vulnerabilities/namespace-view?s[Namespace]=^:name$';
        expect(buildViewLinkUrl(basePath, { name: 'stackrox' }, undefined)).toBe(
            '/main/vulnerabilities/namespace-view?s[Namespace]=^stackrox$'
        );
    });

    it('should combine basePath query and searchParams', () => {
        const basePath = '/main/page?existing=value';
        expect(buildViewLinkUrl(basePath, {}, 'extra=param')).toBe(
            '/main/page?existing=value&extra=param'
        );
    });

    it('should fall back to the path pattern when :param has no value', () => {
        expect(buildViewLinkUrl('/main/risk/:id', {}, undefined)).toBe('/main/risk/:id');
    });

    it('should handle path without any :params or query', () => {
        expect(buildViewLinkUrl('/main/risk', {}, undefined)).toBe('/main/risk');
    });
});
