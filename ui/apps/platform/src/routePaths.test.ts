import { describe, expect, it, vi } from 'vitest';
import { isRouteEnabled } from './routePaths';
import type { IsFeatureFlagEnabled } from 'hooks/useFeatureFlags';
import type { HasReadAccess } from 'hooks/usePermissions';

describe('routePaths', () => {
    describe('isRouteEnabled for base-images', () => {
        it('should enable route when feature flag is enabled and user has Image read access', () => {
            const hasReadAccess: HasReadAccess = vi.fn((resource) => resource === 'Image');
            const isFeatureFlagEnabled: IsFeatureFlagEnabled = vi.fn(
                (flag) => flag === 'ROX_BASE_IMAGE_DETECTION'
            );

            const enabled = isRouteEnabled({ hasReadAccess, isFeatureFlagEnabled }, 'base-images');

            expect(enabled).toBe(true);
            expect(hasReadAccess).toHaveBeenCalledWith('Image');
            expect(isFeatureFlagEnabled).toHaveBeenCalledWith('ROX_BASE_IMAGE_DETECTION');
        });

        it('should disable route when feature flag is disabled', () => {
            const hasReadAccess: HasReadAccess = vi.fn(() => true);
            const isFeatureFlagEnabled: IsFeatureFlagEnabled = vi.fn(() => false);

            const enabled = isRouteEnabled({ hasReadAccess, isFeatureFlagEnabled }, 'base-images');

            expect(enabled).toBe(false);
        });

        it('should disable route when user lacks Image read access', () => {
            const hasReadAccess: HasReadAccess = vi.fn(() => false);
            const isFeatureFlagEnabled: IsFeatureFlagEnabled = vi.fn(() => true);

            const enabled = isRouteEnabled({ hasReadAccess, isFeatureFlagEnabled }, 'base-images');

            expect(enabled).toBe(false);
        });
    });
});
