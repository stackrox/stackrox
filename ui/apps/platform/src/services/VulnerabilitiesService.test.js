import entityTypes from 'constants/entityTypes';
import axios from './instance';
import { suppressVulns } from './VulnerabilitiesService';

vi.mock('./instance', () => ({ default: { patch: vi.fn() } }));

beforeEach(() => {
    vi.clearAllMocks();
});

it.each([
    ['default', [], '0s'],
    ['indefinite', ['0'], '0s'],
    ['finite', ['86400s'], '86400s'],
])('sends the %s node snooze duration', async (_name, args, expected) => {
    const cves = ['CVE-2026-1234'];
    axios.patch.mockResolvedValue({ data: {} });
    await expect(suppressVulns(entityTypes.NODE_CVE, cves, ...args)).resolves.toEqual({});
    expect(axios.patch).toHaveBeenCalledTimes(1);
    expect(axios.patch).toHaveBeenLastCalledWith('/v1/nodecves/suppress', {
        cves,
        duration: expected,
    });
});

it('skips the suppression request for image CVEs', async () => {
    await expect(suppressVulns(entityTypes.IMAGE_CVE, ['CVE-2026-1234'])).resolves.toEqual({});
    expect(axios.patch).not.toHaveBeenCalled();
});
