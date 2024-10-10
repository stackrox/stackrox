import { useEffect } from 'react';
import { useLocation } from 'react-router-dom';

import useAnalytics from './useAnalytics';

function useAnalyticsPageTracking() {
    const { pathname } = useLocation();
    const { analyticsPageVisit } = useAnalytics();

    useEffect(() => {
        analyticsPageVisit('Page Viewed', '', { path: pathname });
    }, [pathname, analyticsPageVisit]);
}

export default useAnalyticsPageTracking;
