import { useEffect } from 'react';
import useAnalytics from 'hooks/useAnalytics';
import { useLocation } from 'react-router-dom-v5-compat';

/*
 * This hook is used to track page views for the plugin.
 */
export function useAnalyticsPageView() {
    const { analyticsPageVisit } = useAnalytics();
    const location = useLocation();

    useEffect(() => {
        analyticsPageVisit('Page Viewed', '', { path: location.pathname });
    }, [analyticsPageVisit, location.pathname]);
}
